package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// A codex ChatGPT login refreshed inside the wrapper, the way codex 0.172 stores it
// (codex-rs/login/src/auth/storage.rs, storage_file.rs, storage_commit.rs): the
// auth.json link in CODEX_HOME is RESOLVED, the commit lock is taken beside the
// resolved file (auth.json.lock), a temp file is written in that directory and
// renamed over the login. A refresh that cannot be saved is an io error, which codex
// reports as `workspace routing discovery unauthorized (401)` -- the incident of
// 2026-11-26 on mediabox.
//
// The fake auth server behind it accepts exactly one refresh token at a time and
// rotates it; presenting any other is reuse, recorded in server/reused. That is
// what a refresh lost, or one seat refreshing with a token another already
// rotated, looks like from the server's side.
const fakeCodex0172 = `#!/bin/sh
set -u
rt() { sed -n 's/.*"refresh_token":"\([^"]*\)".*/\1/p' "$1"; }
locked() {
    l=$1
    shift
    if command -v flock >/dev/null 2>&1; then flock -w 30 "$l" "$@"; else lockf -k -t 30 "$l" "$@"; fi
}
fail() { echo "workspace routing discovery unauthorized (401)"; exit 1; }
if [ "${1-}" = __refresh ]; then
    t=$2
    old=$(rt "$t")
    cur=$(cat "$FAKE_SERVER/current")
    if [ "$old" != "$cur" ]; then echo "$old" >>"$FAKE_SERVER/reused"; exit 1; fi
    [ -z "${FAKE_SLOW:-}" ] || sleep "$FAKE_SLOW"
    n=$((${cur#r} + 1))
    echo "r$n" >"$FAKE_SERVER/current"
    tmp=$(mktemp "${t%/*}/.auth-XXXXXX") || exit 1
    printf '{"tokens":{"account_id":"%s","refresh_token":"r%s"},"last_refresh":"2030-01-01T00:%02d:00Z"}' "${FAKE_ACCT:-acct-1}" "$n" "$n" >"$tmp" || exit 1
    mv -f "$tmp" "$t" 2>/dev/null || { rm -f "$tmp"; exit 1; }
    exit 0
fi
f=$CODEX_HOME/auth.json
[ -e "$f" ] || { echo "not logged in"; exit 1; }
t=$(readlink -f "$f") || fail
echo "LOGIN $(rt "$t")"
if [ -n "${FAKE_REFRESH:-}" ]; then
    locked "${t%.json}.json.lock" /bin/sh "$0" __refresh "$t" || fail
    echo "REFRESHED $(rt "$t")"
fi
if [ -n "${FAKE_EVIL:-}" ]; then
    # A lens that wants out: the real CODEX_HOME's config.toml runs commands the
    # next time the operator starts codex, and a link where the wrapper writes the
    # login would put it wherever the lens likes.
    echo planted >"${t%/*}/../config.toml" 2>/dev/null || true
    rm -f "$t" && ln -s "$PWD/leaked-login" "$t" && echo PLANTED
fi
if [ -n "${FAKE_HANG:-}" ]; then
    echo HANGING
    sleep 60
fi
exit 0
`

type codexLoginRig struct {
	wrapperRig
	server string
}

func newCodexLoginRig(t *testing.T) codexLoginRig {
	t.Helper()
	r := codexLoginRig{wrapperRig: newWrapperRig(t)}
	r.server = filepath.Join(filepath.Dir(r.home), "server")
	if err := os.MkdirAll(r.server, 0o700); err != nil {
		t.Fatal(err)
	}
	r.server = phys(t, r.server)
	if err := os.WriteFile(filepath.Join(r.server, "current"), []byte("r1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.bin, "codex"), []byte(fakeCodex0172), 0o755); err != nil {
		t.Fatal(err)
	}
	r.setLogin(t, "acct-1", "r1", "2026-11-24T17:36:00Z")
	return r
}

func (r codexLoginRig) setLogin(t *testing.T, acct, token, when string) {
	t.Helper()
	body := fmt.Sprintf(`{"tokens":{"account_id":%q,"refresh_token":%q},"last_refresh":%q}`, acct, token, when)
	if err := os.WriteFile(filepath.Join(r.realCodex, "auth.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (r codexLoginRig) cmd(env ...string) *exec.Cmd {
	cmd := exec.Command("/bin/sh", r.wrapper, r.target, "ro", "--", "codex")
	cmd.Dir = r.target
	cmd.Env = append([]string{
		"HOME=" + r.home,
		"TMPDIR=" + r.tmp,
		"PATH=" + r.bin + ":/usr/bin:/bin:/usr/sbin:/sbin",
		"FIXPOINT_SANDBOX_RO=" + r.bin,
		"FIXPOINT_SANDBOX_RW=" + r.server,
		"FAKE_SERVER=" + r.server,
	}, env...)
	return cmd
}

func (r codexLoginRig) seat(t *testing.T, env ...string) string {
	t.Helper()
	out, err := r.cmd(env...).CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper failed: %v\n%s", err, out)
	}
	return string(out)
}

// realToken is the refresh token in the operator's real auth.json.
func (r codexLoginRig) realToken(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.realCodex, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	i := strings.Index(s, `"refresh_token":"`)
	if i < 0 {
		t.Fatalf("no refresh_token in auth.json: %s", s)
	}
	s = s[i+len(`"refresh_token":"`):]
	return s[:strings.IndexByte(s, '"')]
}

func (r codexLoginRig) serverToken(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.server, "current"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

func (r codexLoginRig) noReuse(t *testing.T) {
	t.Helper()
	if b, err := os.ReadFile(filepath.Join(r.server, "reused")); err == nil {
		t.Errorf("the auth server saw rotated refresh tokens presented again: %q", b)
	}
}

func TestBuiltinWrapperLetsCodexSaveARefreshedLogin(t *testing.T) {
	r := newCodexLoginRig(t)
	out := r.seat(t, "FAKE_REFRESH=1")
	if !strings.Contains(out, "REFRESHED r2") {
		t.Fatalf("codex could not save its refreshed login inside the wrapper:\n%s", out)
	}
	if got := r.realToken(t); got != "r2" {
		t.Errorf("real auth.json holds %s after the run, want the refreshed r2", got)
	}
	// And the next run starts from it: no reuse of the rotated r1.
	out = r.seat(t, "FAKE_REFRESH=1")
	if !strings.Contains(out, "LOGIN r2") || !strings.Contains(out, "REFRESHED r3") {
		t.Errorf("the second run did not continue from the refreshed login:\n%s", out)
	}
	r.noReuse(t)
}

// Three seats at once (one review's lenses, or several runs): one login and one
// commit lock for all of them, so each refresh starts from the one before it.
func TestBuiltinWrapperSharesOneLoginAcrossParallelSeats(t *testing.T) {
	r := newCodexLoginRig(t)
	const seats = 3
	outs := make([]string, seats)
	errs := make([]error, seats)
	var wg sync.WaitGroup
	for i := range seats {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b, err := r.cmd("FAKE_REFRESH=1", "FAKE_SLOW=0.3").CombinedOutput()
			outs[i], errs[i] = string(b), err
		}(i)
	}
	wg.Wait()
	for i := range seats {
		if errs[i] != nil {
			t.Errorf("seat %d failed: %v\n%s", i, errs[i], outs[i])
		}
	}
	r.noReuse(t)
	if got := r.serverToken(t); got != "r4" {
		t.Errorf("the server's live token is %s, want r4 (three refreshes in a row)", got)
	}
	if got := r.realToken(t); got != r.serverToken(t) {
		t.Errorf("real auth.json holds %s, the server's live token is %s: the operator's login is stale", got, r.serverToken(t))
	}
}

// fixpoint ends a timed-out or interrupted agent with SIGKILL to its process group,
// so the wrapper's EXIT trap does not run; an operator's kill is a TERM, which it
// does handle. Neither may lose a refresh that codex has already saved.
func TestBuiltinWrapperKeepsARefreshWhenKilled(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		t.Run(sig.String(), func(t *testing.T) {
			r := newCodexLoginRig(t)
			cmd := r.cmd("FAKE_REFRESH=1", "FAKE_HANG=1")
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			pr, pw := io.Pipe()
			cmd.Stdout, cmd.Stderr = pw, pw
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			var mu sync.Mutex
			var b strings.Builder
			seen := func() string {
				mu.Lock()
				defer mu.Unlock()
				return b.String()
			}
			hanging := make(chan bool, 1)
			read := make(chan struct{})
			go func() {
				defer close(read)
				sc := bufio.NewScanner(pr)
				for sc.Scan() {
					mu.Lock()
					b.WriteString(sc.Text() + "\n")
					mu.Unlock()
					if sc.Text() == "HANGING" {
						hanging <- true
					}
				}
				close(hanging)
			}()
			select {
			case ok := <-hanging:
				if !ok {
					_ = cmd.Wait()
					t.Fatalf("the agent exited before its refresh was in place:\n%s", seen())
				}
			case <-time.After(30 * time.Second):
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				t.Fatalf("the agent never got to its refresh:\n%s", seen())
			}
			if sig == syscall.SIGKILL {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Signal(sig)
			}
			_ = cmd.Wait()
			_ = pw.Close()
			<-read
			if !strings.Contains(seen(), "REFRESHED r2") {
				t.Fatalf("codex could not save its refreshed login inside the wrapper:\n%s", seen())
			}
			if sig == syscall.SIGTERM {
				if got := r.realToken(t); got != "r2" {
					t.Errorf("after a TERM the real auth.json holds %s, want r2", got)
				}
			}
			// The next run picks the refresh up, whatever happened to the last one.
			out := r.seat(t, "FAKE_REFRESH=1")
			if !strings.Contains(out, "LOGIN r2") || !strings.Contains(out, "REFRESHED r3") {
				t.Errorf("the next run did not start from the refresh the killed one saved:\n%s", out)
			}
			if got := r.realToken(t); got != "r3" {
				t.Errorf("real auth.json holds %s, want r3", got)
			}
			r.noReuse(t)
		})
	}
}

// A lens that saves a login naming another account must not get it into the
// operator's auth.json, nor into the next seat's.
func TestBuiltinWrapperRefusesASavedLoginForAnotherAccount(t *testing.T) {
	r := newCodexLoginRig(t)
	out := r.seat(t, "FAKE_REFRESH=1", "FAKE_ACCT=attacker")
	if !strings.Contains(out, "REFRESHED") {
		t.Fatalf("the fake could not save its login:\n%s", out)
	}
	if !strings.Contains(out, "different account") {
		t.Errorf("no warning for the refused login:\n%s", out)
	}
	b, err := os.ReadFile(filepath.Join(r.realCodex, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "acct-1") || r.realToken(t) != "r1" {
		t.Errorf("auth.json = %s: another account's login was copied back", b)
	}
	out = r.seat(t)
	if !strings.Contains(out, "LOGIN r1") {
		t.Errorf("the next seat did not get the operator's own login back:\n%s", out)
	}
}

// A login codex outside the sandbox refreshed in the meantime is the newer one,
// and the next seat starts from it rather than from the copy the wrapper kept.
func TestBuiltinWrapperPicksUpALoginRefreshedOutside(t *testing.T) {
	r := newCodexLoginRig(t)
	r.seat(t)
	r.setLogin(t, "acct-1", "r7", "2026-11-26T14:49:29Z")
	if err := os.WriteFile(filepath.Join(r.server, "current"), []byte("r7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := r.seat(t, "FAKE_REFRESH=1")
	if !strings.Contains(out, "LOGIN r7") || !strings.Contains(out, "REFRESHED r8") {
		t.Errorf("the seat did not start from the login refreshed outside:\n%s", out)
	}
	r.noReuse(t)
	if got := r.realToken(t); got != "r8" {
		t.Errorf("real auth.json holds %s, want r8", got)
	}
}

// What a lens can do to the shared directory stays there: it cannot reach the real
// CODEX_HOME through it, and a link it plants is replaced, not written through.
func TestBuiltinWrapperWritesNothingThroughTheSharedLogin(t *testing.T) {
	r := newCodexLoginRig(t)
	out := r.seat(t, "FAKE_EVIL=1")
	if !strings.Contains(out, "PLANTED") {
		t.Fatalf("the fake could not plant its link:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(r.realCodex, "config.toml")); err == nil {
		t.Error("the lens wrote config.toml into the real CODEX_HOME")
	}
	if _, err := os.Lstat(filepath.Join(r.target, "leaked-login")); err == nil {
		t.Error("the wrapper wrote the login through a link the lens planted")
	}
	shared := filepath.Join(r.realCodex, ".fixpoint-sandbox", "auth.json")
	fi, err := os.Lstat(shared)
	if err != nil || !fi.Mode().IsRegular() {
		t.Errorf("the shared login is not a regular file again after the run: %v %v", fi, err)
	}
	if got := r.realToken(t); got != "r1" {
		t.Errorf("real auth.json holds %s, want it untouched (r1)", got)
	}
	out = r.seat(t)
	if !strings.Contains(out, "LOGIN r1") {
		t.Errorf("the next seat did not get the login back:\n%s", out)
	}
}
