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

// Supervise runs cmd to completion under the process-group discipline every
// runner in this program needs -- the agent CLIs, the verify commands, and the
// git/gh subprocesses -- copying its output into stdout and stderr, and returns
// the LEADER's own result. A nil stderr sends both streams to stdout over one
// descriptor (a combined capture). Supervise owns cmd.Stdout, cmd.Stderr,
// cmd.SysProcAttr, cmd.Cancel and cmd.WaitDelay; the caller sets everything
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
// Here the pipes belong to this process, so exec starts no copy goroutine, Wait
// returns on the leader's exit alone, the group is SIGKILLed immediately, and
// only then is the remaining output drained -- with everything that could still
// write into the repository already dead. All cleanup completes before Supervise
// returns: no copy goroutine is left touching stdout or stderr.
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
	// Backstop only: with no exec-owned output pipe left to drain, WaitDelay's
	// remaining job is to kill a leader that ignored the cancel signal, so Wait
	// cannot block forever. (cmd.Stdin may still be an ordinary reader, which exec
	// copies through a pipe of its own, so ErrWaitDelay remains possible; the
	// leaked-pipe handling below covers that too.)
	cmd.WaitDelay = pipeDrainGrace

	outPipe, err := NewOutPipe(stdout)
	if err != nil {
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
			return false, perr
		}
		pipes = append(pipes, errPipe)
		cmd.Stderr = errPipe.ChildFile()
	}

	startErr := cmd.Start()
	// Drop the parent's write ends now the child holds its own: while this process
	// keeps one, the read end can never see EOF.
	for _, p := range pipes {
		p.CloseChild()
	}
	if startErr != nil {
		for _, p := range pipes {
			p.Drain()
		}
		return false, startErr
	}

	waitErr := cmd.Wait()
	// The leader is gone: kill its group BEFORE draining, so nothing that could
	// still touch the repository outlives the command by even the drain.
	_ = KillProcessGroup(cmd)
	// Drain the streams concurrently: a descendant that escaped the group and can
	// outlast the grace usually holds BOTH ends, and one after the other would
	// spend the grace twice for the one thing it is there to bound.
	drained := make(chan bool, len(pipes))
	for _, p := range pipes {
		go func() { drained <- p.Drain() }()
	}
	cut := false
	for range pipes {
		if <-drained {
			cut = true
		}
	}
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
