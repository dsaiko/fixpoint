package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// bundle writes a minimal, valid bundle into dir and returns dir.
func bundle(t *testing.T, dir string, configs map[string]string, prompts, agents []string) string {
	t.Helper()
	for _, sub := range []string{promptsDir, agentsDir} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range configs {
		if err := os.WriteFile(filepath.Join(dir, name+configExt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range prompts {
		if err := os.WriteFile(filepath.Join(dir, promptsDir, p+promptExt), []byte("{{.OutputContract}}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range agents {
		body := "command: [true]\ncan_edit: true\n"
		if err := os.WriteFile(filepath.Join(dir, agentsDir, a+configExt), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// realDir is a temporary directory with every symlink in its path resolved, which
// is the form ProjectRoot returns. The bare t.TempDir() is not always that form --
// on macOS $TMPDIR sits under /var -> private/var -- so an expectation built from
// it would compare two spellings of the same directory.
func realDir(t *testing.T, dir string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// The project root anchors every relative path in a run, so it must be found by
// walking up -- otherwise the same command reviews a different subtree depending
// on which directory it was invoked from.
func TestProjectRootWalksUp(t *testing.T) {
	root := realDir(t, t.TempDir())
	deep := filepath.Join(root, "internal", "target", "sub")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	// ".git" as a FILE, the shape git uses inside a worktree or submodule. A
	// directory-only check would walk straight past the root of every worktree.
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: /elsewhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{root, filepath.Join(root, "internal"), deep} {
		got, err := ProjectRoot(from)
		if err != nil {
			t.Fatal(err)
		}
		if got != root {
			t.Errorf("ProjectRoot(%s) = %s, want %s", from, got, root)
		}
	}
}

// The bundle directory name must NOT be a root marker: "config" is an extremely
// common package name (this repository has internal/config), and treating it as a
// marker detected internal/ as the project root -- so a run from internal/target
// reviewed only that subtree and could not find any config.
func TestProjectRootIgnoresNestedConfigDir(t *testing.T) {
	root := realDir(t, t.TempDir())
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "config")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("ProjectRoot(%s) = %s, want the repository root %s; a nested config package must not look like a project root", nested, got, root)
	}
}

// ~/.fixpoint is the documented USER bundle, so it must not mark the home
// directory as a project root: a non-git project anywhere below home would
// otherwise anchor to the whole home directory -- reviewing $HOME, writing
// artifacts there, and reporting the user's own bundle as project-supplied policy.
func TestProjectRootIgnoresUserBundleInHome(t *testing.T) {
	home := realDir(t, t.TempDir())
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, userBundleDir, promptsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(home, "src", "thing")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(project)
	if err != nil {
		t.Fatal(err)
	}
	if got != project {
		t.Errorf("ProjectRoot(%s) = %s, want %s; the user bundle must not make $HOME a project root", project, got, project)
	}
	// The same directory name IS a marker outside home: it is where a previous
	// run's artifacts land in a project that is not a git repository.
	other := realDir(t, t.TempDir())
	deep := filepath.Join(other, "sub")
	if err := os.MkdirAll(filepath.Join(other, userBundleDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := ProjectRoot(deep); err != nil || got != other {
		t.Errorf("ProjectRoot(%s) = %s, %v; want the artifact directory to mark %s", deep, got, err, other)
	}
}

// Only an artifact DIRECTORY is a marker. A regular file of the same name -- an
// editor's dotfile, a stray note -- must be ignored: honoring it would outrank the
// repository root above it, anchoring the run to a nested subtree and then failing
// to create the default artifact directory because a file already holds the name.
func TestProjectRootIgnoresArtifactFile(t *testing.T) {
	root := realDir(t, t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "target")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, userBundleDir), []byte("note\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(nested)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("ProjectRoot(%s) = %s, want the repository root %s; a regular %s file must not mark a project root", nested, got, root, userBundleDir)
	}
}

// A project that ships its own bundle but is not a git repository still has a
// root -- the documented <project>/config. It is recognized by the bundle's shape,
// never by the bare name, so a nested config package cannot pose as one.
func TestProjectRootFindsNonGitProjectBundle(t *testing.T) {
	root := realDir(t, t.TempDir())
	t.Setenv("HOME", t.TempDir())
	bundle(t, filepath.Join(root, projectBundleDir), map[string]string{"fix-code": "description: x\n"}, []string{"fix"}, nil)
	deep := filepath.Join(root, "internal", "config")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(deep)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("ProjectRoot(%s) = %s, want %s; a non-git project's own bundle marks its root, and a config package without a bundle's shape does not", deep, got, root)
	}
}

// Walking up is lexical, so the walk has to start from the REAL directory: `cd`
// through a symlink leaves $PWD on the link and os.Getwd honors it, so a walk from
// the link's own path climbs the link's parents -- which are not the project's --
// finds no marker, and anchors the run to the linked subtree: a fraction of the
// repository reviewed, artifacts written next to the link.
func TestProjectRootResolvesSymlinkedDir(t *testing.T) {
	root := realDir(t, t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte("gitdir: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "internal", "config")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	// The link lives outside the repository, so nothing on ITS parent chain is a
	// marker -- exactly the case a lexical walk gets wrong.
	link := filepath.Join(realDir(t, t.TempDir()), "cfg")
	if err := os.Symlink(nested, link); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectRoot(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Errorf("ProjectRoot(%s) = %s, want the repository root %s the symlink points into", link, got, root)
	}
}

// A missing name must name every location searched: without that list a user
// cannot tell a typo from a missing bundle from a shadowing surprise.
func TestResolverNotFoundListsSearchPath(t *testing.T) {
	r := &Resolver{Bundles: []string{"/nowhere/a", "/nowhere/b"}}
	_, err := r.Config("ghost")
	if err == nil {
		t.Fatal("expected an error for a missing config")
	}
	for _, want := range []string{"ghost", "/nowhere/a", "/nowhere/b"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q:\n%s", want, err)
		}
	}
}

// Resolution is per file, so a project can shadow one prompt and inherit the
// rest. This is the whole point of a search path -- and the reason every resolved
// path is logged.
func TestResolverShadowsPerFile(t *testing.T) {
	high := bundle(t, filepath.Join(t.TempDir(), "project"), nil, []string{"review-bugs"}, nil)
	low := bundle(t, filepath.Join(t.TempDir(), "system"), nil, []string{"review-bugs", "fix"}, nil)
	r := &Resolver{Bundles: []string{high, low}}

	got, err := r.Prompt("review-bugs")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(high, promptsDir, "review-bugs.md"); got != want {
		t.Errorf("Prompt(review-bugs) = %s, want the shadowing copy %s", got, want)
	}
	// Not shadowed: falls through to the lower bundle.
	got, err = r.Prompt("fix")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(low, promptsDir, "fix.md"); got != want {
		t.Errorf("Prompt(fix) = %s, want the inherited %s", got, want)
	}
}

// A bundle reference is a bare name. The names are chosen by a task config, and
// the first bundle searched is <project>/config -- which the repository under
// review may ship -- so a name carrying "../" would otherwise hand an agent any
// file on the host as its instruction stream.
func TestResolverRejectsNameThatEscapesTheBundle(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{"task": runnableBody}, []string{"fix", "review-bugs"}, []string{"mock"})
	// The traversal lands on a file that really exists, so an unchecked lookup
	// succeeds rather than falling through to a not-found.
	if err := os.WriteFile(filepath.Join(root, "secret.md"), []byte("host secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.yaml"), []byte("description: host secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Resolver{Bundles: []string{dir}}

	if got, err := r.Prompt("../../secret"); err == nil {
		t.Errorf("Prompt(../../secret) resolved to %s; a traversing name must be refused", got)
	}
	if got, err := r.Agent("../../secret"); err == nil {
		t.Errorf("Agent(../../secret) resolved to %s; a traversing name must be refused", got)
	}
	// `extends` is a name a config from the repository under review supplies, so it
	// gets no path escape hatch either -- the base is read, and fragments of it can
	// surface through the decoder's error, before the trust gate can refuse the run.
	if err := os.WriteFile(filepath.Join(dir, "task"+configExt), []byte("extends: ../../secret\n"+taskBody), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBundle(r, "task", root, Overrides{})
	if err == nil {
		t.Fatal("extends naming a path outside the bundle must be refused")
	}
	if !strings.Contains(err.Error(), "separator") {
		t.Errorf("the refusal should say why the name is not usable:\n%v", err)
	}
}

// A bundle file that is a symlink pointing OUT of its bundle is refused. The
// project's bundle is searched first, so following one would let the repository
// under review nominate any file on the host as fixpoint's policy -- and the
// listing is printed before any trust gate can object.
func TestResolverRefusesSymlinkOutOfBundle(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"secret.md":   "host secret",
		"secret.yaml": "description: host secret\n",
	} {
		if err := os.WriteFile(filepath.Join(outside, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	high := bundle(t, filepath.Join(root, "project"), nil, nil, nil)
	low := bundle(t, filepath.Join(root, "system"), map[string]string{"task": runnableBody}, []string{"fix"}, nil)
	link(t, filepath.Join(outside, "secret.md"), filepath.Join(high, promptsDir, "fix"+promptExt))
	link(t, filepath.Join(outside, "secret.yaml"), filepath.Join(high, "task"+configExt))
	r := &Resolver{Bundles: []string{high, low}}

	got, err := r.Prompt("fix")
	if err == nil {
		t.Fatalf("Prompt(fix) resolved to %s, which is a symlink out of the bundle", got)
	}
	if !strings.Contains(err.Error(), "outside its bundle") {
		t.Errorf("the refusal should name the reason:\n%v", err)
	}
	if _, err := r.Config("task"); err == nil {
		t.Fatal("Config(task) must be refused: the first bundle holding it escapes the bundle")
	}
	// The listing omits it instead of failing: it walks every bundle, so one
	// hostile entry must not take `--list` down for all of them -- and the
	// description of a file elsewhere on the host must not appear in it either.
	// The name is dropped rather than shown against the lower bundle's copy,
	// because the escaping entry is refused outright by the by-name lookup: `--list`
	// and completion must not offer a name that every run refuses.
	entries, err := r.ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("--list offers %s -> %s, but a run of it fails with the containment error", e.Name, e.Path)
	}
}

// A bundle entry that is not a regular file -- a dangling symlink, a device --
// is not a match for the by-name lookup, which falls through to the next bundle.
// The listing must fall through with it: skipping the name entirely would hide a
// perfectly good config that a run WOULD use.
func TestListConfigsFallsThroughNonRegularEntry(t *testing.T) {
	root := t.TempDir()
	high := bundle(t, filepath.Join(root, "project"), nil, nil, nil)
	low := bundle(t, filepath.Join(root, "system"), map[string]string{"task": runnableBody}, nil, nil)
	link(t, filepath.Join(root, "nowhere"), filepath.Join(high, "task"+configExt))
	r := &Resolver{Bundles: []string{high, low}}

	want, err := r.Config("task")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := r.ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != want {
		t.Fatalf("--list must show %s, the file a run resolves to: %+v", want, entries)
	}
}

// A bundle entry that is not an ordinary file is not a match. A device or a fifo
// planted through a symlink is worse than a miss: reading /dev/zero never
// returns, and this lookup runs before the trust gate could refuse the target's
// bundle at all.
func TestResolverSkipsNonRegularBundleFile(t *testing.T) {
	root := t.TempDir()
	high := bundle(t, filepath.Join(root, "project"), nil, nil, nil)
	low := bundle(t, filepath.Join(root, "system"), nil, []string{"fix"}, nil)
	link(t, os.DevNull, filepath.Join(high, promptsDir, "fix"+promptExt))

	got, err := (&Resolver{Bundles: []string{high, low}}).Prompt("fix")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(low, promptsDir, "fix"+promptExt); got != want {
		t.Errorf("Prompt(fix) = %s, want the real file %s: a device must never be opened as a bundle file", got, want)
	}
}

// Every read of a bundle file is bounded. `fixpoint --list` reads every config in
// <project>/config to describe it, before any trust gate applies, so an enormous
// file there must not be able to exhaust memory just because someone looked.
func TestBundleFileReadIsCapped(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{"task": runnableBody}, []string{"fix", "review-bugs"}, []string{"mock"})
	huge := append([]byte("description: huge\n"), make([]byte, maxBundleFile)...)
	if err := os.WriteFile(filepath.Join(dir, "huge"+configExt), huge, 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Resolver{Bundles: []string{dir}}

	_, err := LoadBundle(r, "huge", root, Overrides{})
	if err == nil {
		t.Fatal("an oversized config must be refused rather than read")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Errorf("the refusal should say the file is too large:\n%v", err)
	}
	// Listing survives it: an unreadable config is left for the loader to reject,
	// and the configs beside it stay discoverable.
	entries, err := r.ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want both configs listed: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.Name == "huge" && e.Description != "" {
			t.Errorf("nothing should have been read from the oversized config: %q", e.Description)
		}
	}
}

// An unreadable bundle directory -- a mode-000 <project>/config shipped by the
// target, a stale root-owned ~/.fixpoint -- is a PARTIAL failure. The error is
// reported, but the configs the other bundles offered come back with it: listing
// walks every bundle, so one directory nobody can read must not take `--list` and
// completion down for all of them.
func TestListConfigsSurvivesUnreadableBundle(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode, so there is nothing to deny here")
	}
	root := t.TempDir()
	closed := bundle(t, filepath.Join(root, "project"), map[string]string{"hidden": runnableBody}, nil, nil)
	open := bundle(t, filepath.Join(root, "system"), map[string]string{"task": runnableBody}, nil, nil)
	if err := os.Chmod(closed, 0o000); err != nil {
		t.Fatal(err)
	}
	// Restored before TempDir's own cleanup, which cannot remove a directory it is
	// not allowed to read.
	t.Cleanup(func() { _ = os.Chmod(closed, 0o700) })

	got, err := (&Resolver{Bundles: []string{closed, open}}).ListConfigs()
	if err == nil {
		t.Error("an unreadable bundle must still be reported, not swallowed")
	}
	if len(got) != 1 || got[0].Name != "task" {
		t.Fatalf("got %+v, want the readable bundle's config returned alongside the error", got)
	}
}

// link creates a symlink at name pointing at target, replacing whatever is there.
func link(t *testing.T, target, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink(target, name); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// ListConfigs backs both --list and shell completion, so it must report the file a
// run would actually use -- not every copy on the path -- and must say which
// configs are runnable.
func TestListConfigsOmitsShadowedCopies(t *testing.T) {
	high := bundle(t, filepath.Join(t.TempDir(), "project"), map[string]string{"full": runnableBody}, nil, nil)
	low := bundle(t, filepath.Join(t.TempDir(), "system"), map[string]string{
		"full": runnableBody,
		"pr":   runnableBody,
	}, nil, nil)
	r := &Resolver{Bundles: []string{high, low, "/nonexistent"}}
	got, err := r.ListConfigs()
	if err != nil {
		t.Fatalf("a missing bundle on the path must not be an error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d configs, want 2 (a shadowed copy must not be listed twice): %+v", len(got), got)
	}
	byName := map[string]Entry{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if want := filepath.Join(high, "full.yaml"); byName["full"].Path != want {
		t.Errorf("full resolved to %s, want the shadowing %s", byName["full"].Path, want)
	}
	if want := filepath.Join(low, "pr.yaml"); byName["pr"].Path != want {
		t.Errorf("pr resolved to %s, want %s", byName["pr"].Path, want)
	}
}

// A base config -- one with no review lenses, existing to be inherited via
// `extends` -- is listed but marked non-runnable. Listing it matters for
// discovery: you cannot write `extends: defaults` without knowing it is there.
// Marking it matters because an unmarked listing reads as "things you can run".
//
// The distinction is by SHAPE, not by the name "defaults": a bundle may hold
// several bases under any names.
func TestListConfigsMarksBaseConfigs(t *testing.T) {
	dir := bundle(t, filepath.Join(t.TempDir(), "b"), map[string]string{
		"task":        runnableBody,
		"defaults":    "loop:\n  max_iterations: 3\n",
		"shared-base": "target: {mode: directory}\n",
	}, nil, nil)
	got, err := (&Resolver{Bundles: []string{dir}}).ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	runnable := map[string]bool{}
	for _, c := range got {
		runnable[c.Name] = c.Runnable
	}
	if !runnable["task"] {
		t.Error("a config with review lenses must be runnable")
	}
	for _, base := range []string{"defaults", "shared-base"} {
		if runnable[base] {
			t.Errorf("%q defines no lenses and must be marked non-runnable", base)
		}
	}
	if len(got) != 3 {
		t.Errorf("got %d configs, want all 3 listed -- bases are marked, not hidden", len(got))
	}
}

// A config that inherits its lenses -- and its description -- from an `extends`
// base is runnable, because the run merges the child over the base. Judging it on
// its own text alone would print a working config as a non-runnable base and hide
// it from shell completion entirely.
func TestListConfigsResolvesExtendsForRunnability(t *testing.T) {
	dir := bundle(t, filepath.Join(t.TempDir(), "b"), map[string]string{
		"defaults":   "description: shared base\n" + runnableBody,
		"inherits":   "extends: defaults\nloop:\n  max_iterations: 2\n",
		"own-desc":   "description: mine\nextends: defaults\n",
		"no-base":    "extends: missing\n",
		"plain-base": "loop:\n  max_iterations: 3\n",
	}, nil, nil)
	got, err := (&Resolver{Bundles: []string{dir}}).ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Entry{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if !byName["inherits"].Runnable {
		t.Error("a config inheriting its lenses via extends must be runnable, not reported as a base")
	}
	if got, want := byName["inherits"].Description, "shared base"; got != want {
		t.Errorf("inherited description = %q, want %q", got, want)
	}
	if got, want := byName["own-desc"].Description, "mine"; got != want {
		t.Errorf("own description = %q, want %q -- the child's own value wins", got, want)
	}
	if !byName["no-base"].Runnable {
		// The loader reports the unresolvable base by name; the listing must not
		// silently reclassify the config as something to inherit from.
		t.Error("a config whose extends base does not resolve must still be offered, not marked a base")
	}
	if byName["plain-base"].Runnable {
		t.Error("a config with no lenses and no extends is still a base")
	}
}

// An unparseable or partially invalid config must not vanish from the listing:
// listing is discovery, and the loader is where correctness is reported with a
// message that says what is wrong.
func TestListConfigsKeepsUnparseableConfigs(t *testing.T) {
	dir := bundle(t, filepath.Join(t.TempDir(), "b"), map[string]string{
		"broken": "roles: [this is not a mapping",
	}, nil, nil)
	got, err := (&Resolver{Bundles: []string{dir}}).ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Runnable {
		t.Errorf("an unparseable config must still be listed and left for the loader to reject: %+v", got)
	}
}

const runnableBody = taskBody

const taskBody = `roles:
  coder: {agent: mock, prompt: fix}
  review:
    strategy: fixed
    prompts:
      - {agent: mock, prompt: review-bugs}
`

// extends must override per key and leave untouched keys inherited, and a list the
// child sets must REPLACE the inherited one -- appending would make an inherited
// entry impossible to remove.
func TestLoadBundleExtends(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"defaults": "target:\n  mode: directory\n  exclude: [\"**/vendor/**\", \"**/testdata/**\"]\nloop:\n  max_iterations: 9\n  clean_rounds_to_stop: 3\n",
		"task":     "extends: defaults\nloop:\n  max_iterations: 2\ntarget:\n  exclude: [\"only/**\"]\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg := l.Config
	if cfg.Loop.MaxIterations != 2 {
		t.Errorf("MaxIterations = %d, want the child's 2", cfg.Loop.MaxIterations)
	}
	if cfg.Loop.CleanRoundsToStop != 3 {
		t.Errorf("CleanRoundsToStop = %d, want the inherited 3", cfg.Loop.CleanRoundsToStop)
	}
	if cfg.Target.Mode != ModeDirectory {
		t.Errorf("Mode = %q, want the inherited directory", cfg.Target.Mode)
	}
	if len(cfg.Target.Exclude) != 1 || cfg.Target.Exclude[0] != "only/**" {
		t.Errorf("Exclude = %v, want the child's list to replace the base's", cfg.Target.Exclude)
	}
	if l.Source.Extends == "" {
		t.Error("Source.Extends must record the base file for provenance")
	}
}

// A config must not be able to assert its own trust. Bundles resolve from
// <project>/config FIRST, so the file under test here is one a reviewed repository
// can ship: honoring a trust key in it would let hostile code authorize both
// executing the agent definitions it supplies and running the write-capable coder
// against it -- defeating the two gates that exist for exactly that case.
//
// Every field is checked in both positions (the task config and the base it
// extends), because inheritance would otherwise be the way around the rule.
func TestLoadBundleRejectsSelfGrantedTrust(t *testing.T) {
	for _, key := range []string{"trusted_target", "trusted_bundle", "allow_untrusted_fix"} {
		t.Run(key+"/direct", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"task": "target: {mode: directory}\nloop:\n  " + key + ": true\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			_, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
			if err == nil {
				t.Fatalf("loading a config that sets loop.%s must fail: reviewed code could authorize itself", key)
			}
			if !strings.Contains(err.Error(), "cannot be set in a configuration file") {
				t.Errorf("the error must explain the boundary, got: %v", err)
			}
		})
		// Setting it to false is refused too. Allowing the key with a "safe" value
		// would mean the loader has to be right about which values are safe, and a
		// reader of the config would reasonably conclude the key works.
		t.Run(key+"/false-is-also-refused", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"task": "target: {mode: directory}\nloop:\n  " + key + ": false\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			if _, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{}); err == nil {
				t.Fatalf("loop.%s: false must also be refused, so the key never looks supported", key)
			}
		})
		t.Run(key+"/via-extends", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"base": "target: {mode: directory}\nloop:\n  " + key + ": true\n",
				"task": "extends: base\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			if _, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{}); err == nil {
				t.Fatalf("loop.%s must be refused in an inherited base too, or extends is the way around the rule", key)
			}
		})
	}
}

// The trust fields must not be reachable through YAML at all -- the test above
// asserts the friendly refusal, this one asserts the type itself cannot carry the
// value, which is what makes silent acceptance impossible if rejectTrustKeys is
// ever bypassed or removed.
func TestTrustFieldsAreNotYAMLDecodable(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte("loop:\n  trusted_target: true\n  trusted_bundle: true\n  allow_untrusted_fix: true\n"), &cfg); err != nil {
		t.Fatalf("permissive decode should not error here: %v", err)
	}
	if cfg.Loop.TrustedTarget || cfg.Loop.TrustedBundle || cfg.Loop.AllowUntrustedFix {
		t.Error("YAML set a trust field; these must be settable only by the CLI flags")
	}
}

// The same boundary as the two tests above, for the publishing switches. Posting
// is an action on somebody else's pull request, and <project>/config is searched
// FIRST -- so a review.post key a repository could ship would let the code under
// review have a review, and with post_verdict an APPROVE, published on its own
// pull request under the operator's identity, with no flag from the operator.
//
// Both positions are checked, since inheritance would otherwise be the way around
// the rule, and false is refused as well: allowing a "safe" value would mean the
// loader has to be right about which values are safe, and a reader of the config
// would reasonably conclude the key works.
func TestLoadBundleRejectsSelfGrantedPosting(t *testing.T) {
	for _, key := range []string{"post", "post_verdict"} {
		t.Run(key+"/direct", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"task": "target: {mode: directory}\nreview:\n  " + key + ": true\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			_, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
			if err == nil {
				t.Fatalf("loading a config that sets review.%s must fail: reviewed code could publish on its own pull request", key)
			}
			if !strings.Contains(err.Error(), "cannot be set in a configuration file") {
				t.Errorf("the error must explain the boundary, got: %v", err)
			}
		})
		t.Run(key+"/false-is-also-refused", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"task": "target: {mode: directory}\nreview:\n  " + key + ": false\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			if _, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{}); err == nil {
				t.Fatalf("review.%s: false must also be refused, so the key never looks supported", key)
			}
		})
		t.Run(key+"/via-extends", func(t *testing.T) {
			root := t.TempDir()
			dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
				"base": "target: {mode: directory}\nreview:\n  " + key + ": true\n",
				"task": "extends: base\n" + taskBody,
			}, []string{"fix", "review-bugs"}, []string{"mock"})
			if _, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{}); err == nil {
				t.Fatalf("review.%s must be refused in an inherited base too, or extends is the way around the rule", key)
			}
		})
	}
}

// And the type itself must not carry the value, for the same reason the trust
// fields must not: this is what keeps silent acceptance impossible if the probe or
// KnownFields(true) is ever bypassed. A `yaml:"post"` tag added in a later change
// fails here rather than shipping a config-settable publish switch.
func TestPostingFieldsAreNotYAMLDecodable(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte("review:\n  post: true\n  post_verdict: true\n"), &cfg); err != nil {
		t.Fatalf("permissive decode should not error here: %v", err)
	}
	if cfg.Review.Post || cfg.Review.PostVerdict {
		t.Errorf("YAML set a publishing flag (post=%v post_verdict=%v); these must be settable only by -post / -post-verdict",
			cfg.Review.Post, cfg.Review.PostVerdict)
	}
}

// Inheritance is one level deep: a chain would mean the effective value of a
// field requires reading N files, defeating the point of naming the base.
func TestLoadBundleRejectsExtendsChain(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"a": "extends: b\n" + taskBody,
		"b": "extends: c\n",
		"c": "target: {mode: directory}\n",
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	_, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "a", root, Overrides{})
	if err == nil || !strings.Contains(err.Error(), "one level deep") {
		t.Fatalf("expected a one-level-deep error, got %v", err)
	}
}

// Every relative path resolves against the project root, so a run from a
// subdirectory reviews the whole project and writes artifacts at the top -- not
// into whatever directory it happened to start in.
func TestLoadBundleAnchorsPathsToProjectRoot(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target:\n  mode: directory\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if l.Config.Target.Path != root {
		t.Errorf("Target.Path = %s, want the project root %s", l.Config.Target.Path, root)
	}
	// The logs default is applied before anchoring; anchoring after is what keeps
	// the artifact directory at the project root rather than the working directory.
	if !strings.HasPrefix(l.Config.Logs.Dir, root) {
		t.Errorf("Logs.Dir = %s, want it anchored under %s", l.Config.Logs.Dir, root)
	}
}

// Agents load from their own files so a task config says which agents it uses
// without restating how to invoke them.
func TestLoadBundleResolvesAgentsFromFiles(t *testing.T) {
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target: {mode: directory}\n" + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})

	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := l.Config.Agents["mock"]
	if !ok {
		t.Fatal("agent mock was not loaded from agents/mock.yaml")
	}
	if a.PromptVia != PromptViaStdin {
		t.Errorf("PromptVia = %q, want the default %q applied to file-loaded agents too", a.PromptVia, PromptViaStdin)
	}
	if a.Timeout.Std() == 0 {
		t.Error("Timeout default was not applied to a file-loaded agent")
	}
	if l.Source.Agents["mock"] == "" {
		t.Error("Source.Agents must record where each agent resolved from")
	}
}

// loadTask builds a one-config bundle and compiles it with ov, which is the whole
// shape of the override tests below: a config the flags then act on.
func loadTask(t *testing.T, body string, ov Overrides) *Loaded {
	t.Helper()
	root := t.TempDir()
	dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
		"task": "target: {mode: directory}\n" + body + taskBody,
	}, []string{"fix", "review-bugs"}, []string{"mock"})
	l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "task", root, ov)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// Overrides are part of compiling the configuration, not something a caller does
// to the result. This is the property that makes Loaded immutable: if LoadBundle
// did not apply them, every caller would have to mutate Config afterwards and
// re-validate in the right order.
func TestLoadBundleAppliesOverrides(t *testing.T) {
	l := loadTask(t, "loop:\n  max_iterations: 3\n", Overrides{
		ReviewOnly:        true,
		MaxIterations:     7,
		AllowUntrustedFix: true,
		TrustedTarget:     true,
	})
	if !l.Config.Loop.ReviewOnly {
		t.Error("ReviewOnly was not applied by LoadBundle")
	}
	if !l.Config.Loop.AllowUntrustedFix {
		t.Error("AllowUntrustedFix was not applied by LoadBundle")
	}
	if !l.Config.Loop.TrustedTarget {
		t.Error("TrustedTarget was not applied by LoadBundle")
	}
	if l.Config.Loop.MaxIterations != 7 {
		t.Errorf("MaxIterations = %d, want the flag's 7 to beat the config's 3", l.Config.Loop.MaxIterations)
	}
	if l.Overrides.MaxIterations != 7 {
		t.Error("Loaded.Overrides must record the assertions, so a run can report what came from a flag")
	}
}

// The booleans force ON only. A zero-valued Overrides must never turn off a gate
// the config set deliberately -- otherwise merely omitting a flag would weaken a
// configured run, and the flags are assertions the operator adds, not a full
// description of the run.
func TestLoadBundleOverridesOnlyForceOn(t *testing.T) {
	l := loadTask(t, "loop:\n  review_only: true\n  max_iterations: 3\n", Overrides{})
	if !l.Config.Loop.ReviewOnly {
		t.Error("an absent -review-only flag turned off the config's review_only")
	}
	if l.Config.Loop.MaxIterations != 3 {
		t.Errorf("MaxIterations = %d, want the config's 3 kept when the flag is 0 (unset)", l.Config.Loop.MaxIterations)
	}
}

// A negative -max-iterations must reach Validate and be rejected there, rather than
// being swallowed as "no flag supplied". Zero is the documented "use config"
// sentinel, so only zero may be ignored.
func TestLoadBundleNegativeMaxIterationsReachesValidate(t *testing.T) {
	l := loadTask(t, "loop:\n  max_iterations: 3\n", Overrides{MaxIterations: -1})
	if l.Config.Loop.MaxIterations != -1 {
		t.Fatalf("MaxIterations = %d, want the invalid -1 preserved for Validate to reject", l.Config.Loop.MaxIterations)
	}
	err := l.Validate()
	if err == nil || !strings.Contains(err.Error(), "max_iterations") {
		t.Fatalf("Validate() = %v, want a max_iterations rejection", err)
	}
	// Validate names the task config: with extends and a per-file search path, the
	// rule that failed does not identify the file to edit.
	if !strings.Contains(err.Error(), l.Source.Config) {
		t.Errorf("Validate() error must name the config file %s, got: %v", l.Source.Config, err)
	}
}

// The bundled review-pr config carries no PR number, because which pull request to
// review is decided per invocation. Without an override that config is not runnable
// at all -- Validate rejects a pr-mode config whose number is not positive -- so the
// flag has to reach Target.PR before validation, not after it.
func TestLoadBundlePROverrideMakesPRModeRunnable(t *testing.T) {
	const prBody = "target:\n  mode: pr\n  pr: 0\nloop:\n  review_only: true\n" +
		"roles:\n  coder: {agent: mock, prompt: fix}\n" +
		"  review:\n    strategy: fixed\n    prompts:\n      - {agent: ro, prompt: review-bugs}\n"
	load := func(ov Overrides) *Loaded {
		t.Helper()
		root := t.TempDir()
		dir := bundle(t, filepath.Join(root, projectBundleDir), map[string]string{
			"review-pr": prBody,
		}, []string{"fix", "review-bugs"}, []string{"mock"})
		// Reviewers must be read-only, which the shared bundle helper's agents are not.
		if err := os.WriteFile(filepath.Join(dir, agentsDir, "ro"+configExt), []byte("command: [true]\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		l, err := LoadBundle(&Resolver{Bundles: []string{dir}}, "review-pr", root, ov)
		if err != nil {
			t.Fatal(err)
		}
		return l
	}

	l := load(Overrides{})
	if err := l.Validate(); err == nil || !strings.Contains(err.Error(), "target.pr") {
		t.Fatalf("Validate() = %v, want the placeholder pr: 0 rejected", err)
	}

	l = load(Overrides{PR: 1234})
	if l.Config.Target.PR != 1234 {
		t.Errorf("Target.PR = %d, want the flag's 1234 to replace the placeholder", l.Config.Target.PR)
	}
	if err := l.Validate(); err != nil {
		t.Errorf("Validate() = %v, want the config to be runnable once -pr supplied it a number", err)
	}
}

// Applied is what the run log reports, so it must name every asserted override and
// stay quiet when none were made.
func TestOverridesApplied(t *testing.T) {
	if got := (Overrides{}).Applied(); len(got) != 0 {
		t.Errorf("Applied() = %v, want empty for a run with no flags", got)
	}
	got := strings.Join(Overrides{
		ReviewOnly:        true,
		MaxIterations:     -1,
		PR:                1234,
		AllowUntrustedFix: true,
		TrustedTarget:     true,
	}.Applied(), ", ")
	for _, want := range []string{"review_only=true", "allow_untrusted_fix=true", "trusted_target=true", "max_iterations=-1", "pr=1234"} {
		if !strings.Contains(got, want) {
			t.Errorf("Applied() = %q, missing %q", got, want)
		}
	}
}

// A name that looks like a path is used directly, so an ad-hoc config outside any
// bundle still runs.
func TestResolverAcceptsPathLikeNames(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "adhoc.yaml")
	if err := os.WriteFile(p, []byte("target: {mode: directory}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Resolver{Bundles: []string{"/nowhere"}}
	got, err := r.Config(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != p {
		t.Errorf("Config(%s) = %s, want the path itself", p, got)
	}
}
