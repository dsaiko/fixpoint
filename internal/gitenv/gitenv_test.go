package gitenv

import (
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
