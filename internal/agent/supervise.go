package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// pipeDrainGrace bounds how long Supervise waits for a child's output pipes to
// drain AFTER the leader has exited and its process group has been killed.
// Everything still in that group is dead by then, so the write ends are closed
// and the drain finishes at once; the grace exists only for a descendant that
// escaped the group (setsid/setpgrp) and holds a write end open, which would
// otherwise wedge the run for as long as that process lives.
const pipeDrainGrace = 2 * time.Second

// OutPipe carries one of a child process's output streams into dst through a
// pipe whose read end THIS process owns.
//
// Handing exec an *os.File rather than an ordinary writer is the whole point:
// exec then passes the descriptor to the child directly and starts no copy
// goroutine of its own, so cmd.Wait returns the moment the LEADER exits instead
// of when the last descendant holding the write end closes it. That separation
// is what lets a caller SIGKILL the process group immediately on the leader's
// exit -- before draining -- rather than after cmd.WaitDelay has elapsed. See
// Supervise for why the ordering matters.
//
// It is exported for target's streaming git listing, which consumes stdout
// itself and needs the same discipline for stderr.
type OutPipe struct {
	child *os.File // write end the child inherits
	r     *os.File // read end this process owns
	done  chan struct{}
}

// NewOutPipe returns a pipe that copies everything the child writes into dst.
// The caller assigns ChildFile to cmd.Stdout and/or cmd.Stderr, calls
// CloseChild once the command has started (or failed to start), and Drain once
// it has exited.
func NewOutPipe(dst io.Writer) (*OutPipe, error) {
	if dst == nil {
		dst = io.Discard
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create output pipe: %w", err)
	}
	p := &OutPipe{child: w, r: r, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		_, _ = io.Copy(dst, r)
	}()
	return p, nil
}

// ChildFile is the write end to hand to exec. Passing one OutPipe's ChildFile
// as both cmd.Stdout and cmd.Stderr gives the child a single descriptor for
// both streams, so a combined capture keeps the child's own interleaving.
func (p *OutPipe) ChildFile() *os.File { return p.child }

// CloseChild drops this process's copy of the write end. It must be called once
// the command has started: while the parent still holds a write end no EOF can
// ever arrive, so every Drain would burn the full grace.
func (p *OutPipe) CloseChild() { _ = p.child.Close() }

// Drain waits for the copy goroutine to finish and reports whether the capture
// was cut short rather than ending at a real EOF.
//
// When it returns, nothing is still writing to dst: a caller may read dst
// without racing the copy. If the grace expires -- some process outside the
// killed group still holds a write end -- closing the read end unblocks the
// copy's read, which is the only thing it can be waiting on (dst is an
// in-memory buffer that never blocks), so the wait that follows returns at
// once. That is a wait, not an abandonment; the guarantee above holds on both
// paths.
func (p *OutPipe) Drain() bool {
	select {
	case <-p.done:
		_ = p.r.Close()
		return false
	case <-time.After(pipeDrainGrace):
	}
	_ = p.r.Close()
	<-p.done
	return true
}

// inPipe feeds a caller-supplied reader into a child through a pipe whose write
// end THIS process owns -- the mirror image of OutPipe, and there for the same
// reason.
//
// With an ordinary reader as cmd.Stdin, os/exec makes the pipe itself and runs
// the copy in a goroutine cmd.Wait waits for, bounded only by cmd.WaitDelay. A
// prompt larger than the pipe buffer whose reader never drains it -- an agent
// that spawns a child which inherits fd 0, then exits without consuming the rest
// of the prompt -- leaves that copy blocked in write(2), so cmd.Wait cannot
// return until the delay expires and the process-group kill is postponed by
// exactly the window that kill exists to close. Handing exec an *os.File instead
// means it starts no stdin goroutine, so cmd.Wait returns on the leader's exit
// alone.
type inPipe struct {
	child *os.File // read end the child inherits
	w     *os.File // write end this process owns
	done  chan struct{}
}

// newInPipe returns a pipe already copying src towards the child. The caller
// assigns child to cmd.Stdin, calls closeChild once the command has started (or
// failed to start), and join once the process group is dead.
func newInPipe(src io.Reader) (*inPipe, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	p := &inPipe{child: r, w: w, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		_, _ = io.Copy(w, src)
		// EOF on the child's stdin: an agent that reads its prompt to the end waits
		// for this before it starts work.
		_ = w.Close()
	}()
	return p, nil
}

// closeChild drops this process's copy of the read end. It must be called once
// the command has started: while the parent still holds one, a child that never
// read the input cannot fail the feed with EPIPE, so join would burn the full
// grace instead of returning at once.
func (p *inPipe) closeChild() { _ = p.child.Close() }

// join waits for the feed goroutine to finish, so nothing is still reading the
// caller's reader when Supervise returns. A feed cut short is ordinary -- an
// agent is free to stop reading its prompt -- so this reports nothing; it exists
// only to end the goroutine. Once the process group is dead every read end is
// closed and a blocked write fails immediately; the grace covers a descendant
// that escaped the group and still holds one, and closing the write end unblocks
// the copy so the wait that follows returns at once.
func (p *inPipe) join() {
	select {
	case <-p.done:
		return
	case <-time.After(pipeDrainGrace):
	}
	_ = p.w.Close()
	<-p.done
}

// endFeed stops a stdin feed and waits for it, so nothing is left reading the
// caller's reader once Supervise returns. A nil pipe means stdin was never ours.
func endFeed(p *inPipe) {
	if p == nil {
		return
	}
	p.closeChild()
	p.join()
}

// drainAll finishes every pipe Supervise owns and reports whether an output
// capture was cut short. It runs them concurrently because a descendant that
// escaped the process group and can outlast the grace usually holds every end it
// inherited, and one after the other would spend the grace once per end. The
// stdin feed reports nothing: an agent is free to stop reading its prompt, so a
// feed that ended early is ordinary rather than a cut capture.
func drainAll(pipes []*OutPipe, stdinPipe *inPipe) bool {
	done := make(chan bool, len(pipes)+1)
	waiting := 0
	for _, p := range pipes {
		waiting++
		go func() { done <- p.Drain() }()
	}
	if stdinPipe != nil {
		waiting++
		go func() { stdinPipe.join(); done <- false }()
	}
	cut := false
	for range waiting {
		if <-done {
			cut = true
		}
	}
	return cut
}

// Supervise runs cmd to completion under the process-group discipline every
// runner in this program needs -- the agent CLIs, the verify commands, and the
// git/gh subprocesses -- copying its output into stdout and stderr, and returns
// the LEADER's own result. A nil stderr sends both streams to stdout over one
// descriptor (a combined capture). Supervise owns cmd.Stdout, cmd.Stderr,
// cmd.SysProcAttr, cmd.Cancel and cmd.WaitDelay, and takes over delivery of a
// cmd.Stdin the caller set as an ordinary reader; the caller sets everything
// else (Dir, Env, Stdin) before calling.
//
// It exists because the ORDER of teardown matters and cmd.Run gets it wrong for
// our purposes. With an ordinary writer as cmd.Stdout, os/exec copies the output
// through a pipe IT owns, and cmd.Run cannot return until that copy sees EOF or
// cmd.WaitDelay expires. A leader that exits 0 after backgrounding a child which
// inherited the pipe therefore holds cmd.Run for the whole WaitDelay -- and only
// once cmd.Run returns can the caller kill the process group. For those two
// seconds a descendant of a command that has already finished is still live
// inside the target repository: it can edit a tracked file after the check that
// would have caught the edit has already been recorded as passing, and the round
// then commits work no check ever saw.
//
// Here every pipe belongs to this process -- stdin's as much as the output ones,
// since exec's stdin copy is a goroutine cmd.Wait waits for just the same -- so
// exec starts no copy goroutine, Wait returns on the leader's exit alone, the
// group is SIGKILLed immediately, and only then is the remaining output drained,
// with everything that could still write into the repository already dead. All
// cleanup completes before Supervise returns: no copy goroutine is left touching
// stdout or stderr, or reading the caller's stdin.
//
// leakedPipe reports that the capture may be cut short: some process outside the
// group held a write end past the drain grace, or exec itself reported a drain
// timeout. Callers note that in what they persist. err is the LEADER's: a drain
// timeout for a leader that exited 0 is not turned into a failure, for the
// reasons SucceededDespiteLeakedPipe gives.
func Supervise(ctx context.Context, cmd *exec.Cmd, stdout, stderr io.Writer) (leakedPipe bool, err error) {
	if cmd.Stdout != nil || cmd.Stderr != nil {
		return false, errors.New("agent.Supervise: cmd.Stdout and cmd.Stderr must be unset; Supervise owns the output pipes")
	}
	// Own the whole subprocess lifecycle, not just the leader: these CLIs spawn
	// children (an MCP server, a compiler, a test daemon, a credential helper, a
	// commit hook), and killing only the leader leaves them running -- free to edit
	// the repository concurrently with the next check, the clean-tree check or the
	// round commit, or to leak past an aborted run.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return KillProcessGroup(cmd) }
	// Backstop only: with no pipe of exec's own left -- neither output nor stdin --
	// WaitDelay's remaining job is to kill a leader that ignored the cancel signal,
	// so Wait cannot block forever.
	cmd.WaitDelay = pipeDrainGrace

	// Take over stdin unless exec hands it to the child as a descriptor and runs no
	// copy goroutine of its own: a nil Stdin (which exec makes /dev/null) or an
	// *os.File the caller opened.
	var stdinPipe *inPipe
	if _, isFile := cmd.Stdin.(*os.File); cmd.Stdin != nil && !isFile {
		p, perr := newInPipe(cmd.Stdin)
		if perr != nil {
			return false, perr
		}
		cmd.Stdin = p.child
		stdinPipe = p
	}

	outPipe, err := NewOutPipe(stdout)
	if err != nil {
		endFeed(stdinPipe)
		return false, err
	}
	pipes := []*OutPipe{outPipe}
	cmd.Stdout = outPipe.ChildFile()
	cmd.Stderr = outPipe.ChildFile()
	if stderr != nil {
		errPipe, perr := NewOutPipe(stderr)
		if perr != nil {
			outPipe.CloseChild()
			outPipe.Drain()
			endFeed(stdinPipe)
			return false, perr
		}
		pipes = append(pipes, errPipe)
		cmd.Stderr = errPipe.ChildFile()
	}

	startErr := cmd.Start()
	// Drop the parent's copies of the ends the child now holds: while this process
	// keeps an output write end, the read end can never see EOF, and while it keeps
	// the stdin read end, a feed nobody consumes can never fail.
	for _, p := range pipes {
		p.CloseChild()
	}
	if stdinPipe != nil {
		stdinPipe.closeChild()
	}
	if startErr != nil {
		drainAll(pipes, stdinPipe)
		return false, startErr
	}

	waitErr := cmd.Wait()
	// The leader is gone: kill its group BEFORE draining, so nothing that could
	// still touch the repository outlives the command by even the drain.
	_ = KillProcessGroup(cmd)
	cut := drainAll(pipes, stdinPipe)
	if SucceededDespiteLeakedPipe(cmd, waitErr) {
		waitErr = nil
		cut = true
	}
	if ctx.Err() != nil {
		// Torn down by Ctrl-C or by the command's own timeout. The caller reports
		// that, and a capture cut short by the teardown itself needs no separate note.
		cut = false
	}
	return cut, waitErr
}
