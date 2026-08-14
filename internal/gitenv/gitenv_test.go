package gitenv

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The -c form and the environment form must express the SAME pins. Two callers
// depend on them being interchangeable -- fixpoint's own git command lines take the
// flags, the agent CLIs and gh's internal git calls only ever see the environment --
// so a key added to one shape and not the other is a hole that looks closed.
func TestBothShapesCarryEveryPin(t *testing.T) {
	args := SafeConfigArgs()
	env := Harden([]string{})
	for i, kv := range safeConfig {
		if !slices.Contains(args, "-c") || !slices.Contains(args, kv[0]+"="+kv[1]) {
			t.Errorf("SafeConfigArgs() = %v, want it to pin %s=%s", args, kv[0], kv[1])
		}
		n := strconv.Itoa(i)
		if !slices.Contains(env, "GIT_CONFIG_KEY_"+n+"="+kv[0]) ||
			!slices.Contains(env, "GIT_CONFIG_VALUE_"+n+"="+kv[1]) {
			t.Errorf("Harden() = %v, want it to pin %s=%s as entry %s", env, kv[0], kv[1], n)
		}
	}
	if want := "GIT_CONFIG_COUNT=" + strconv.Itoa(len(safeConfig)); !slices.Contains(env, want) {
		t.Errorf("Harden() = %v, want %s", env, want)
	}
	// The settings the guard exists for, spelled out: a rename or a dropped entry
	// should fail here rather than silently in production.
	for _, want := range []string{"core.hooksPath=/dev/null", "core.fsmonitor=false", "protocol.ext.allow=never", "core.alternateRefsCommand=false"} {
		if !slices.Contains(args, want) {
			t.Errorf("SafeConfigArgs() = %v, want it to pin %s", args, want)
		}
	}
}

// SafeConfigArgs returns a fresh slice, because every caller appends its own
// arguments to it. A shared backing array would let one git command's args land in
// the next one's.
func TestSafeConfigArgsIsNotShared(t *testing.T) {
	mine := append(SafeConfigArgs(), "status")
	if got := SafeConfigArgs(); slices.Contains(got, "status") {
		t.Errorf("SafeConfigArgs() = %v, want it unaffected by a caller's append to %v", got, mine)
	}
}

// nil means "the invoking process's environment", NOT "an empty environment": the
// result goes to cmd.Env, where a nil would mean "inherit, unhardened" -- the exact
// hole Harden closes for an inherit_all agent.
func TestHardenNilExpandsTheParentEnvironment(t *testing.T) {
	t.Setenv("GITENV_MARKER", "present")
	env := Harden(nil)
	if !slices.Contains(env, "GITENV_MARKER=present") {
		t.Errorf("Harden(nil) dropped the parent environment: %v", env)
	}
	if got := Harden([]string{}); len(got) != 2*len(safeConfig)+1 {
		t.Errorf("Harden([]) = %v, want only the pins: an empty env must stay empty", got)
	}
}

// A caller that legitimately set its own overrides keeps them: ours are appended
// after theirs and the count is merged, rather than clobbering entry 0 and leaving
// the caller's own entries above a count that no longer covers them.
func TestHardenMergesAnExistingCount(t *testing.T) {
	env := Harden([]string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=user.name",
		"GIT_CONFIG_VALUE_0=caller",
	})
	for _, want := range []string{
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=caller",
		"GIT_CONFIG_KEY_1=core.hooksPath",
		"GIT_CONFIG_COUNT=" + strconv.Itoa(1+len(safeConfig)),
	} {
		if !slices.Contains(env, want) {
			t.Errorf("Harden() = %v, want %s", env, want)
		}
	}
	if n := strings.Count(strings.Join(env, "\n"), "GIT_CONFIG_COUNT="); n != 1 {
		t.Errorf("Harden() = %v, want exactly one GIT_CONFIG_COUNT, got %d", env, n)
	}
}

// A count git could not use is not a reason to number our entries past it: ours
// start at 0 so the block stays self-consistent.
func TestHardenIgnoresAnUnusableCount(t *testing.T) {
	for _, bad := range []string{"", "nonsense", "-3", "0"} {
		env := Harden([]string{"GIT_CONFIG_COUNT=" + bad})
		if !slices.Contains(env, "GIT_CONFIG_KEY_0=core.hooksPath") {
			t.Errorf("Harden(count=%q) = %v, want our pins to start at 0", bad, env)
		}
		if want := "GIT_CONFIG_COUNT=" + strconv.Itoa(len(safeConfig)); !slices.Contains(env, want) {
			t.Errorf("Harden(count=%q) = %v, want %s", bad, env, want)
		}
	}
}

// NoOperatorConfig is the i12/i27 control: with the operator's global and system
// git config switched off there is nowhere left to define the filter a
// coder-authored .gitattributes names. It had no direct test -- only indirect
// coverage through implement.Git.run -- so a regression that let
// GIT_CONFIG_GLOBAL survive would have re-opened the hole (review run
// 20260814-012440).
func TestNoOperatorConfigPinsTheOperatorsFilesOff(t *testing.T) {
	want := map[string]string{
		"GIT_CONFIG_GLOBAL": "/dev/null",
		"GIT_CONFIG_SYSTEM": "/dev/null",
		"GIT_ATTR_NOSYSTEM": "1",
	}
	// A caller's own values for these must be REPLACED, not appended to: two
	// entries for one variable is undefined behavior across libc's.
	got := NoOperatorConfig([]string{
		"PATH=/usr/bin",
		"GIT_CONFIG_GLOBAL=/home/me/.gitconfig",
		"GIT_ATTR_NOSYSTEM=0",
	})
	counts := map[string]int{}
	for _, e := range got {
		if k, v, ok := strings.Cut(e, "="); ok {
			if w, pinned := want[k]; pinned {
				counts[k]++
				if v != w {
					t.Errorf("%s = %q, want %q", k, v, w)
				}
			}
		}
	}
	for k := range want {
		if counts[k] != 1 {
			t.Errorf("%s appears %d times, want exactly 1", k, counts[k])
		}
	}
	if !slices.Contains(got, "PATH=/usr/bin") {
		t.Error("an unrelated variable was dropped")
	}

	// nil means the process environment, hardened -- the same rule Harden uses,
	// because the result is assigned to cmd.Env where nil would mean "inherit".
	t.Setenv("GIT_CONFIG_GLOBAL", "/home/me/.gitconfig")
	for _, e := range NoOperatorConfig(nil) {
		if e == "GIT_CONFIG_GLOBAL=/home/me/.gitconfig" {
			t.Error("nil env kept the operator's global config")
		}
	}
}

// PinTools fixes git/gh/glab at the start of a run, so a PATH entry that
// appears inside the target mid-run cannot become the answer (review run
// 20260814-012440). Tested with a stand-in name, because pinning the real ones
// would leak into every other test in the package.
func TestPinnedToolSurvivesAPathChange(t *testing.T) {
	dir := t.TempDir()
	name := "git"
	binary := filepath.Join(dir, name)
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	PinTools()
	t.Cleanup(PinTools) // leave the package's view matching the real PATH
	if first := Tool(name); first != binary {
		t.Fatalf("Tool(%q) = %q, want the pinned %q", name, first, binary)
	}

	// A directory prepended to PATH afterwards -- the shape of a checkout adding
	// bin/git during a run -- must not change the answer.
	hijack := t.TempDir()
	if err := os.WriteFile(filepath.Join(hijack, name), []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", hijack+string(os.PathListSeparator)+dir)
	if again := Tool(name); again != binary {
		t.Errorf("Tool(%q) = %q after a mid-run PATH change, want the pinned %q", name, again, binary)
	}
}

// Before pinning, Tool resolves LIVE rather than caching whatever the first
// caller happened to see: a cache filled by whoever asked first would answer for
// a moment nobody chose.
func TestUnpinnedToolResolvesLive(t *testing.T) {
	dir := t.TempDir()
	name := "fixpoint-test-tool"
	binary := filepath.Join(dir, name)
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if got := Tool(name); got != binary {
		t.Fatalf("Tool(%q) = %q, want %q", name, got, binary)
	}
	other := t.TempDir()
	moved := filepath.Join(other, name)
	if err := os.WriteFile(moved, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", other)
	if got := Tool(name); got != moved {
		t.Errorf("Tool(%q) = %q, want the live %q", name, got, moved)
	}

	// An unresolvable name comes back BARE -- never <cwd>/name, which is a file
	// the reviewed checkout may own when fixpoint runs from inside it. Caught by
	// this assertion the first time it was written.
	if got := Tool("fixpoint-no-such-binary"); got != "fixpoint-no-such-binary" {
		t.Errorf("unresolvable name = %q, want it returned unchanged", got)
	}
}
