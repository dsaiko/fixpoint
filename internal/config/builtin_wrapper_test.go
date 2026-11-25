package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The shipped wrapper, run through the host's real sandbox: bubblewrap on Linux,
// sandbox-exec on macOS. Skipped where neither can confine anything (no bwrap, or
// no user namespaces for it), because a probe that runs unconfined proves nothing.
//
// The agent is a probe script named `codex`, so the wrapper treats it as codex and
// gives it codex's state. It reports what it can reach; each test asserts on that.

type wrapperRig struct {
	wrapper, home, tmp, target, bin, realCodex string
}

func newWrapperRig(t *testing.T) wrapperRig {
	t.Helper()
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("bwrap"); err != nil {
			t.Skip("bwrap not installed")
		}
		if out, err := exec.Command("bwrap", "--ro-bind", "/", "/", "--unshare-pid", "--dev", "/dev", "--proc", "/proc", "true").CombinedOutput(); err != nil {
			t.Skipf("bwrap cannot make a sandbox here: %v %s", err, out)
		}
	case "darwin":
		if _, err := exec.LookPath("sandbox-exec"); err != nil {
			t.Skip("sandbox-exec not available")
		}
	default:
		t.Skip("the wrapper supports Linux and macOS only")
	}
	wrapper, err := filepath.Abs(filepath.Join("..", "..", projectBundleDir, filepath.FromSlash(builtinSandboxRel)))
	if err != nil {
		t.Fatal(err)
	}
	phys := func(p string) string {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatal(err)
		}
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// On Linux the wrapper replaces /tmp, so a fake home under it would vanish for
	// the wrong reason and a missing $HOME mask would go unnoticed: the rig goes
	// under the real home's cache instead (never the fake home, which is what is
	// being hidden).
	base := t.TempDir()
	if runtime.GOOS == "linux" {
		cache, err := os.UserCacheDir()
		if err != nil {
			t.Skipf("no cache directory for the rig: %v", err)
		}
		if err := os.MkdirAll(cache, 0o700); err != nil {
			t.Fatal(err)
		}
		//nolint:usetesting // t.TempDir is under /tmp, which the wrapper replaces; see above
		if base, err = os.MkdirTemp(cache, "fixpoint-wrapper-test."); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(base) })
	}
	root := phys(base)
	r := wrapperRig{
		wrapper: wrapper,
		home:    phys(filepath.Join(root, "home")),
		tmp:     phys(filepath.Join(root, "tmp")),
		target:  phys(filepath.Join(root, "target")),
		bin:     phys(filepath.Join(root, "bin")),
	}
	r.realCodex = phys(filepath.Join(r.home, ".codex"))
	write := func(p, body string, mode os.FileMode) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(r.home, ".ssh", "id_ed25519"), "SECRET-SSH\n", 0o600)
	write(filepath.Join(r.realCodex, "sessions", "old.jsonl"), "SECRET-TRANSCRIPT\n", 0o600)
	write(filepath.Join(r.realCodex, "auth.json"), `{"tokens":{"account_id":"acct-1","refresh_token":"r1"}}`, 0o600)
	write(filepath.Join(r.target, "code.go"), "package main\n", 0o644)
	write(filepath.Join(r.home, ".claude", "settings.json"), "CLAUDE-SETTINGS\n", 0o600)
	write(filepath.Join(r.home, ".claude", "projects", "p", "t.jsonl"), "SECRET-CLAUDE-TRANSCRIPT\n", 0o600)
	write(filepath.Join(r.home, ".claude.json"), "{}\n", 0o600)
	write(filepath.Join(r.home, "proj", "code.go"), "package inhome\n", 0o644)
	write(filepath.Join(r.bin, "claude"), `#!/bin/sh
cat code.go
cat "$HOME/.claude/settings.json" 2>/dev/null && echo CFG-OK
cat "$HOME/.claude/projects/p/t.jsonl" 2>/dev/null && echo LEAK-claude-transcript
cat "$HOME/.codex/auth.json" 2>/dev/null && echo LEAK-codex-auth
cat "$HOME/.ssh/id_ed25519" 2>/dev/null && echo LEAK-ssh
echo '{"n":1}' > "$HOME/.claude.json" && echo CJSON-OK
exit 0
`, 0o755)
	write(filepath.Join(r.bin, "codex"), `#!/bin/sh
cat code.go
cat "$HOME/.ssh/id_ed25519" 2>/dev/null && echo LEAK-ssh
cat "${PROBE_TRANSCRIPT:-$HOME/.codex/sessions/old.jsonl}" 2>/dev/null && echo LEAK-transcript
echo w > "$PWD/written" 2>/dev/null && echo WROTE-target
echo w > "$HOME/planted" 2>/dev/null && [ -e "$HOME/planted" ] && echo WROTE-home-file
if [ -e "$CODEX_HOME/auth.json" ]; then echo "AUTH $(cat "$CODEX_HOME/auth.json")"; else echo NO-AUTH; fi
echo s > "$TMPDIR/scratch" && echo SCRATCH-OK
if [ -n "${REFRESH:-}" ]; then rm -f "$CODEX_HOME/auth.json"; printf '%s' "$REFRESH" > "$CODEX_HOME/auth.json"; fi
exit 0
`, 0o755)
	return r
}

func (r wrapperRig) run(t *testing.T, mode string, extraEnv []string, args ...string) string {
	t.Helper()
	return r.runAs(t, r.target, "codex", mode, extraEnv, args...)
}

// runAs runs the probe named cli (codex or claude) confined to target.
func (r wrapperRig) runAs(t *testing.T, target, cli, mode string, extraEnv []string, args ...string) string {
	t.Helper()
	argv := append([]string{r.wrapper, target, mode, "--", cli}, args...)
	cmd := exec.Command("/bin/sh", argv...)
	cmd.Dir = target
	cmd.Env = append([]string{
		"HOME=" + r.home,
		"TMPDIR=" + r.tmp,
		"PATH=" + r.bin + ":/usr/bin:/bin:/usr/sbin:/sbin",
		// The probe lives outside $HOME and, on Linux, under the /tmp the wrapper
		// replaces: this is the operator's way to hand a CLI its install directory.
		"FIXPOINT_SANDBOX_RO=" + r.bin,
	}, extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestBuiltinWrapperConfinesAReadOnlyAgent(t *testing.T) {
	r := newWrapperRig(t)
	out := r.run(t, "ro", nil)
	if !strings.Contains(out, "package main") {
		t.Fatalf("the agent could not read the directory it reviews:\n%s", out)
	}
	leaks := []string{"LEAK-ssh", "LEAK-transcript", "WROTE-target"}
	// On macOS a write under $HOME is denied. On Linux $HOME is an empty tmpfs that
	// is gone when the agent is, so a write there succeeds and reaches nothing: what
	// matters is the real directory, checked below.
	if runtime.GOOS == "darwin" {
		leaks = append(leaks, "WROTE-home-file")
	}
	for _, leak := range leaks {
		if strings.Contains(out, leak) {
			t.Errorf("%s:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "SCRATCH-OK") {
		t.Errorf("TMPDIR is not writable:\n%s", out)
	}
	if !strings.Contains(out, "acct-1") {
		t.Errorf("a ChatGPT-login codex route was not given its login:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(r.home, "planted")); err == nil {
		t.Error("the agent created a file in the real home directory")
	}
	if ents, _ := os.ReadDir(r.tmp); len(ents) != 0 {
		t.Errorf("scratch left behind in TMPDIR: %v", ents)
	}
}

func TestBuiltinWrapperLetsAnEditingAgentWriteTheTarget(t *testing.T) {
	r := newWrapperRig(t)
	out := r.run(t, "rw", nil)
	if !strings.Contains(out, "WROTE-target") {
		t.Fatalf("rw mode could not write the target:\n%s", out)
	}
	if strings.Contains(out, "LEAK-ssh") {
		t.Errorf("rw mode exposed the home directory:\n%s", out)
	}
}

// A route with a model_provider of its own authenticates with that provider's key
// and must not be handed the operator's OpenAI login.
func TestBuiltinWrapperWithholdsTheLoginFromAProviderRoute(t *testing.T) {
	r := newWrapperRig(t)
	out := r.run(t, "ro", nil, "-c", "model_provider=ollama-cloud")
	if !strings.Contains(out, "NO-AUTH") || strings.Contains(out, "acct-1") {
		t.Fatalf("a provider route saw the ChatGPT login:\n%s", out)
	}
}

func TestBuiltinWrapperPersistsARefreshedLoginForTheSameAccount(t *testing.T) {
	r := newWrapperRig(t)
	refreshed := `{"tokens":{"account_id":"acct-1","refresh_token":"r2"}}`
	r.run(t, "ro", []string{"REFRESH=" + refreshed})
	got, err := os.ReadFile(filepath.Join(r.realCodex, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != refreshed {
		t.Errorf("auth.json = %s, want the refreshed login copied back", got)
	}
}

// What a lens writes into its scratch CODEX_HOME must not become the operator's
// login: that would log them into whoever's account the lens chose.
func TestBuiltinWrapperRefusesALoginForAnotherAccount(t *testing.T) {
	r := newWrapperRig(t)
	out := r.run(t, "ro", []string{"REFRESH=" + `{"tokens":{"account_id":"attacker","refresh_token":"x"}}`})
	got, err := os.ReadFile(filepath.Join(r.realCodex, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "acct-1") {
		t.Errorf("auth.json = %s: another account's login was copied back", got)
	}
	if !strings.Contains(out, "different account") {
		t.Errorf("no warning for the refused login:\n%s", out)
	}
}

func TestBuiltinWrapperRefusesTheHomeDirectoryAsTarget(t *testing.T) {
	r := newWrapperRig(t)
	cmd := exec.Command("/bin/sh", r.wrapper, r.home, "ro", "--", "true")
	cmd.Env = []string{"HOME=" + r.home, "TMPDIR=" + r.tmp, "PATH=/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "home directory") {
		t.Fatalf("err = %v, out = %s; want the home directory refused", err, out)
	}
}

// CODEX_HOME relocated outside $HOME (a direnv setup does this) is not covered by
// hiding $HOME: its transcripts must be masked on their own, and its login still
// handed over.
func TestBuiltinWrapperHidesRelocatedCodexState(t *testing.T) {
	r := newWrapperRig(t)
	moved := filepath.Join(filepath.Dir(r.home), "codexhome")
	if err := os.Rename(r.realCodex, moved); err != nil {
		t.Fatal(err)
	}
	r.realCodex = moved
	out := r.run(t, "ro", []string{
		"CODEX_HOME=" + moved,
		"PROBE_TRANSCRIPT=" + filepath.Join(moved, "sessions", "old.jsonl"),
	})
	if strings.Contains(out, "LEAK-transcript") {
		t.Errorf("relocated CODEX_HOME transcripts were readable:\n%s", out)
	}
	if !strings.Contains(out, "acct-1") {
		t.Errorf("relocated CODEX_HOME login was not handed over:\n%s", out)
	}
}

// The usual layout: the project lives under $HOME. On macOS that means the target
// is allowed back from the $HOME read deny, which is exactly where an allow that
// names the wrong operation silently loses.
func TestBuiltinWrapperReadsATargetInsideHome(t *testing.T) {
	r := newWrapperRig(t)
	out := r.runAs(t, filepath.Join(r.home, "proj"), "codex", "ro", nil)
	if !strings.Contains(out, "package inhome") {
		t.Fatalf("the agent could not read a target inside $HOME:\n%s", out)
	}
	for _, leak := range []string{"LEAK-ssh", "LEAK-transcript", "WROTE-target"} {
		if strings.Contains(out, leak) {
			t.Errorf("%s:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "acct-1") {
		t.Errorf("the codex login was not handed over:\n%s", out)
	}
}

func TestBuiltinWrapperGivesClaudeItsConfigButNotItsTranscripts(t *testing.T) {
	r := newWrapperRig(t)
	out := r.runAs(t, filepath.Join(r.home, "proj"), "claude", "ro", nil)
	if !strings.Contains(out, "package inhome") {
		t.Fatalf("claude could not read its target:\n%s", out)
	}
	if !strings.Contains(out, "CFG-OK") {
		t.Errorf("claude could not read its own config directory:\n%s", out)
	}
	if !strings.Contains(out, "CJSON-OK") {
		t.Errorf("claude could not rewrite .claude.json:\n%s", out)
	}
	for _, leak := range []string{"LEAK-claude-transcript", "LEAK-codex-auth", "LEAK-ssh"} {
		if strings.Contains(out, leak) {
			t.Errorf("%s:\n%s", leak, out)
		}
	}
}

// outsideRig adds the layout the $HOME tmpfs alone does not cover: a symlink in
// $HOME leading out of it to a tree of repositories, the reviewed one among
// them, as `~/oddin -> /tank/enc/oddin` does on a real box. The probe is an
// unknown CLI, so it gets no CLI state.
type outsideRig struct {
	wrapperRig
	repos, a, b, x string
}

func newOutsideRig(t *testing.T) outsideRig {
	t.Helper()
	r := newWrapperRig(t)
	outside := filepath.Join(filepath.Dir(r.home), "outside")
	o := outsideRig{
		wrapperRig: r,
		repos:      filepath.Join(outside, "repos"),
		a:          filepath.Join(outside, "repos", "a"),
		b:          filepath.Join(outside, "repos", "b"),
		x:          filepath.Join(outside, "x"),
	}
	for _, d := range []string{o.a, o.b, o.x} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for p, body := range map[string]string{
		filepath.Join(o.a, "code.go"): "package a\n",
		filepath.Join(o.b, "secret"):  "SECRET-B\n",
		filepath.Join(o.x, "secret"):  "SECRET-X\n",
		filepath.Join(r.bin, "probe"): `#!/bin/sh
cat code.go
cat "$PROBE_B/secret" 2>/dev/null && echo LEAK-b
ls "$PROBE_B" >/dev/null 2>&1 && echo LIST-b
cat "$PROBE_X/secret" 2>/dev/null && echo LEAK-x
(echo w > "$PWD/written") 2>/dev/null && echo WROTE-a
exit 0
`,
	} {
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(o.repos, filepath.Join(r.home, "repos")); err != nil {
		t.Fatal(err)
	}
	return o
}

func (o outsideRig) probe(t *testing.T, mode string, extraEnv ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", o.wrapper, o.a, mode, "--", "probe")
	cmd.Dir = o.a
	cmd.Env = append([]string{
		"HOME=" + o.home,
		"TMPDIR=" + o.tmp,
		"PATH=" + o.bin + ":/usr/bin:/bin:/usr/sbin:/sbin",
		"FIXPOINT_SANDBOX_RO=" + o.bin,
		"PROBE_B=" + o.b,
		"PROBE_X=" + o.x,
	}, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// A tree reached through a symlink in $HOME is the operator's as much as $HOME
// is: the other repositories in it are hidden, while the reviewed one, inside
// that tree, stays readable (and writable in rw only).
func TestBuiltinWrapperHidesTreesSymlinkedFromHome(t *testing.T) {
	o := newOutsideRig(t)
	for _, mode := range []string{"ro", "rw"} {
		out, err := o.probe(t, mode)
		if err != nil {
			t.Fatalf("%s: wrapper failed: %v\n%s", mode, err, out)
		}
		if !strings.Contains(out, "package a") {
			t.Errorf("%s: the reviewed directory inside the hidden tree is not readable:\n%s", mode, out)
		}
		for _, leak := range []string{"LEAK-b", "LIST-b"} {
			if strings.Contains(out, leak) {
				t.Errorf("%s: %s -- another repository beside the target is visible:\n%s", mode, leak, out)
			}
		}
		if wrote := strings.Contains(out, "WROTE-a"); wrote != (mode == "rw") {
			t.Errorf("%s: write to the target = %v:\n%s", mode, wrote, out)
		}
	}
}

// FIXPOINT_SANDBOX_HIDE hides what no symlink in $HOME leads to.
func TestBuiltinWrapperHidesTheOperatorsExtraPaths(t *testing.T) {
	o := newOutsideRig(t)
	out, err := o.probe(t, "ro", "FIXPOINT_SANDBOX_HIDE="+o.x)
	if err != nil {
		t.Fatalf("wrapper failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "LEAK-x") {
		t.Errorf("a FIXPOINT_SANDBOX_HIDE path is readable:\n%s", out)
	}
	if !strings.Contains(out, "package a") {
		t.Errorf("the target is not readable:\n%s", out)
	}
}

// A hide list that cannot be honored stops the agent before it starts: a path
// that does not exist, and one that holds the target (hiding it would hide the
// work, and binding the target back would make the entry mean nothing).
func TestBuiltinWrapperRefusesAHideListItCannotHonor(t *testing.T) {
	o := newOutsideRig(t)
	for name, hide := range map[string]string{
		"missing":  filepath.Join(o.x, "nope"),
		"ancestor": o.repos,
		"target":   o.a,
	} {
		out, err := o.probe(t, "ro", "FIXPOINT_SANDBOX_HIDE="+hide)
		if err == nil {
			t.Errorf("%s: FIXPOINT_SANDBOX_HIDE=%s was accepted:\n%s", name, hide, out)
		}
		if strings.Contains(out, "package a") {
			t.Errorf("%s: the agent ran:\n%s", name, out)
		}
	}
}

// What is bound back on top of a hidden tree: a toolchain whose directory in
// $HOME is a symlink into one, and the agent's own install directory inside one.
// Hiding either would make the agent fail to start rather than leak.
func TestBuiltinWrapperBindsBackToolchainsAndTheAgentInsideAHiddenTree(t *testing.T) {
	o := newOutsideRig(t)
	cargo := filepath.Join(o.repos, "..", "cargo")
	tools := filepath.Join(o.repos, "..", "tools", "bin")
	for _, d := range []string{cargo, tools} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	cargo, tools = phys(t, cargo), phys(t, tools)
	if err := os.WriteFile(filepath.Join(cargo, "tool"), []byte("CARGO-OK\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tools, "inhidden"), []byte("#!/bin/sh\ncat \"$HOME/.cargo/tool\"\ncat "+filepath.Join(cargo, "tool")+" | sed s/CARGO/REAL/\ncat \"$PROBE_B/secret\" 2>/dev/null && echo LEAK-b\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for link, to := range map[string]string{".cargo": cargo, "tools": filepath.Dir(tools)} {
		if err := os.Symlink(to, filepath.Join(o.home, link)); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("/bin/sh", o.wrapper, o.a, "ro", "--", filepath.Join(tools, "inhidden"))
	cmd.Dir = o.a
	cmd.Env = []string{"HOME=" + o.home, "TMPDIR=" + o.tmp, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", "PROBE_B=" + o.b}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("an agent installed inside a hidden tree did not start: %v\n%s", err, out)
	}
	for _, want := range []string{"CARGO-OK", "REAL-OK"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing %s: a symlinked toolchain was not bound back:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "LEAK-b") {
		t.Errorf("binding the agent back exposed the rest of the tree:\n%s", out)
	}
}

func phys(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
