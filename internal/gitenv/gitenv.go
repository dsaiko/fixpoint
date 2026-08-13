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

// operatorConfigOff is the set that takes the OPERATOR's git configuration out
// of play, as opposed to the repository's. safeConfig pins keys by name, which
// only works for keys whose names are static; the dangerous ones here have
// dynamic names -- filter.<name>.clean/smudge, diff.<driver>.command -- so they
// cannot be pinned, only starved of a place to be defined. Switching the global
// and system files off does exactly that: a .gitattributes in the tree names a
// filter, and the definition it needs is no longer readable.
//
// GIT_ATTR_NOSYSTEM completes it by ignoring the system-wide attributes file,
// so the tree's own .gitattributes is the only one in effect.
var operatorConfigOff = []string{
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_CONFIG_SYSTEM=/dev/null",
	"GIT_ATTR_NOSYSTEM=1",
}

// NoOperatorConfig returns env with the operator's global and system git config
// switched off, replacing any values env already carries for those variables so
// the result is unambiguous whatever the caller passed.
//
// Only ever right for a repository FIXPOINT created: nothing there needs the
// operator's identity or aliases, because Init writes that repository's identity
// locally. Over an operator's own repository it would change what their git
// does, which is why this is a named helper and not part of Harden.
//
// It lives beside Harden for the reason Harden's own comment gives: two callers
// that cannot import each other need the same pins, and the second copy is the
// bug. The write path (internal/target.Collector, staging and committing) and
// the read path (internal/implement's status, clone and checkout) both need
// these, and shipped with only the write path covered -- review run
// 20260813-180828 found the read path's `git clone` executing a
// globally-defined filter with fixpoint's credentials in its environment.
func NoOperatorConfig(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	out := make([]string, 0, len(env)+len(operatorConfigOff))
	for _, e := range env {
		if isOperatorConfigVar(e) {
			continue
		}
		out = append(out, e)
	}
	return append(out, operatorConfigOff...)
}

func isOperatorConfigVar(entry string) bool {
	for _, pin := range operatorConfigOff {
		name, _, _ := strings.Cut(pin, "=")
		if strings.HasPrefix(entry, name+"=") {
			return true
		}
	}
	return false
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
