// Package gitenv holds the git-configuration hardening that every subprocess
// fixpoint starts against a target must carry.
//
// It is its own package because two callers that cannot import each other need
// the SAME pins, and a second copy of the list is the bug: internal/target
// prepends them to fixpoint's own git/gh commands as -c overrides, and
// internal/agent exports them into the environment of the reviewer/coder CLIs,
// which run their own `git status`/`git diff`/`git log` inside the target and
// would otherwise honor whatever the target's .git/config says.
package gitenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// safeConfig neutralizes the repo-controlled git settings that make git itself
// execute attacker code. A target's .git is not always a clean `git clone` -- an
// extracted archive or a crafted checkout can ship its own .git/hooks and
// .git/config -- and running git there would otherwise run those programs with
// the environment of whichever process invoked git (fixpoint's own
// ANTHROPIC_API_KEY / GITHUB_TOKEN, or an agent's declared credential), even in a
// review-only run that never passes the fix-round trust gate. core.hooksPath is
// pointed at a directory that cannot hold an executable hook, and core.fsmonitor
// (a command status/diff/ls-files would spawn) is disabled. This does not cover
// gitattributes-driven filters/diff-drivers, which have dynamic names and are
// refused by target.unsafeConfigKey instead; a truly untrusted .git should still
// be reviewed under an external sandbox.
//
// protocol.ext.allow=never kills the ext:: helper protocol, whose URL IS a shell
// command git runs. git's own default for it is already "never", but that default
// is CONFIGURABLE: a crafted .git/config setting protocol.ext.allow=always turns
// the transport back on for exactly the user-initiated fetches fixpoint performs
// (`gh pr checkout`, the base-object fetch), and remote.<name>.url or a repo-local
// url.<ext-url>.insteadOf then routes an ordinary-looking remote into it.
// target.unsafeConfigKey refuses such a rewrite but does not flag
// remote.<name>.url; this pin beats the repo's value outright and closes the whole
// class rather than one key at a time -- including the shapes that reach git
// through gh's internal calls, since Harden exports these as GIT_CONFIG_*.
//
// core.alternateRefsCommand is another value git runs THROUGH THE SHELL. Its
// documentation describes it as server-side only, but that is not where it fires
// for us: whenever the repo has a .git/objects/info/alternates entry, the CLIENT
// side of a fetch enumerates the alternate's tips to seed negotiation, and runs
// this command instead of git-for-each-ref to do it. A crafted checkout that
// ships an alternates file plus this setting would therefore execute it during
// pr mode's `gh pr checkout` or Prepare's base-object fetch. Pinning it to
// `false` (the same shape as core.fsmonitor) makes the enumeration produce no
// tips, which only costs negotiation hints, never correctness.
//
// Every key here is STATIC, which is what makes pinning the right answer for it:
// the settings whose names are dynamic (filter.<name>.clean, diff.<driver>.command,
// ...) cannot be overridden by key and are refused by target.unsafeConfigKey.
var safeConfig = [][2]string{
	{"core.hooksPath", "/dev/null"},
	{"core.fsmonitor", "false"},
	{"protocol.ext.allow", "never"},
	{"core.alternateRefsCommand", "false"},
}

// SafeConfigArgs returns the pins as git's own `-c key=value` flags, for a git
// command line fixpoint builds itself. They beat any value in the target's
// .git/config. It returns a fresh slice, so a caller may append its own args to
// it.
func SafeConfigArgs() []string {
	out := make([]string, 0, 2*len(safeConfig))
	for _, kv := range safeConfig {
		out = append(out, "-c", kv[0]+"="+kv[1])
	}
	return out
}

// Harden returns env plus GIT_CONFIG_COUNT/GIT_CONFIG_KEY_n/GIT_CONFIG_VALUE_n
// entries that apply safeConfig to EVERY git process started with it, including
// ones fixpoint never builds a command line for: the git calls gh makes
// internally, and the `git status`/`git diff` an agent CLI runs while exploring
// the target. git reads these env entries exactly as if they were -c overrides.
//
// A nil env means "the invoking process's own environment" (os.Environ), because
// the result is what a caller assigns to cmd.Env and a nil there would mean
// "inherit, unhardened" -- which is exactly the hole this exists to close. A
// non-nil empty env stays empty apart from the pins, preserving exec's
// empty-environment-vs-inherit distinction.
//
// Any GIT_CONFIG_COUNT already in env is merged rather than clobbered: its
// existing KEY/VALUE entries are preserved and ours are appended after them, so a
// caller that legitimately set its own overrides keeps them.
func Harden(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	base := 0
	out := make([]string, 0, len(env)+2*len(safeConfig)+1)
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, "GIT_CONFIG_COUNT="); ok {
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				base = n
			}
			continue // dropped; re-added below with the merged count
		}
		out = append(out, e)
	}
	n := base
	for _, kv := range safeConfig {
		out = append(out,
			fmt.Sprintf("GIT_CONFIG_KEY_%d=%s", n, kv[0]),
			fmt.Sprintf("GIT_CONFIG_VALUE_%d=%s", n, kv[1]),
		)
		n++
	}
	out = append(out, fmt.Sprintf("GIT_CONFIG_COUNT=%d", n))
	return out
}
