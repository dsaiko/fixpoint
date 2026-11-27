#!/bin/sh
# Tests for bench/sandbox.sh, the filesystem boundary around a bench lens. Run by
# `make bench-check`.
#
# The wrapper is copied into a scratch tree that stands in for the checkout, so
# "the checkout" is a directory this test owns and plants an answer key in. The
# refusals, the fail-closed cases and the Linux argument assembly (bwrap and uname
# stubbed) run everywhere. The CONFINEMENT probe runs through the real sandbox of
# the host it is on: sandbox-exec on macOS, bwrap on Linux; where bwrap is missing
# or cannot make a namespace it is skipped with a printed note and proves nothing.
#
# Every probe has a positive control: the same probe, run unconfined, must find
# its sentinels readable, so a denial is a denial and not a missing file.
#
# SANDBOX_SH=<path> tests another copy of the wrapper: that is how each assertion
# was mutation-checked (break the wrapper, see this test fail, restore).
set -eu

here=$(cd "$(dirname "$0")" && pwd)
SANDBOX_SH=${SANDBOX_SH:-$here/sandbox.sh}
os=$(uname -s)

# base holds the fake HOME and the fake checkout. On Linux the wrapper replaces
# /tmp, so base must NOT be under it (a fake home there would vanish for the wrong
# reason): it goes under the real home's cache. On macOS the only writable root is
# the wrapper's own scratch directory, which is never base.
if [ "$os" = Linux ]; then
    mkdir -p "$HOME/.cache"
    base=$(mktemp -d "$HOME/.cache/fixpoint-sandbox-test.XXXXXX")
else
    base=$(mktemp -d)
fi
base=$(cd "$base" && pwd -P)
trap 'rm -rf "$base"' EXIT

fails=0
fail() {
    echo "FAIL $1"
    fails=$((fails + 1))
}

# reap <pid> <seconds>: wait for a background job, but never longer than the deadline
# (a wrapper that ignores its signal must fail the test, not hang it). Sets reap_rc, and
# reap_late=1 when the deadline had to kill the job.
reap() {
    rm -f "$base/dog-fired"
    ( sleep "$2"; if kill -0 "$1" 2>/dev/null; then : >"$base/dog-fired"; kill -KILL "$1" 2>/dev/null; fi ) >/dev/null 2>&1 &
    dog=$!
    set +e
    wait "$1" 2>/dev/null
    reap_rc=$?
    set -e
    { kill "$dog"; wait "$dog"; } 2>/dev/null || true
    reap_late=0
    [ ! -e "$base/dog-fired" ] || reap_late=1
}

# expect <label> <file> <grep -E pattern> [!]: the file must (or, with !, must
# not) match.
expect() {
    if grep -Eq -- "$3" "$2"; then hit=1; else hit=0; fi
    if [ "${4:-}" = "!" ]; then hit=$((1 - hit)); fi
    [ "$hit" = 1 ] || fail "$1"
}

# scratch_count <dir>: how many wrapper scratch directories <dir> holds. Every run
# below gets a TMPDIR of its own, so the count is this test's alone and a sandbox run
# by someone else on the same host cannot skew it.
scratch_count() { ls "$1" 2>/dev/null | grep -c '^bench-sandbox\.' || true; }

# ---- the stand-in world -------------------------------------------------------
tree=$base/tree
home=$base/home
mkdir -p "$tree/bench" "$tree/.fixpoint/run" "$home/.local/bin" "$base/realfp"
# ~/.fixpoint is a SYMLINK to a directory elsewhere: the profile must hide the real one.
ln -s "$base/realfp" "$home/.fixpoint"
mkdir -p "$home/.codex/sessions" "$home/.claude/projects/p" "$home/.claude/sessions" "$home/.claude/shell-snapshots"
cp "$SANDBOX_SH" "$tree/bench/sandbox.sh"
chmod +x "$tree/bench/sandbox.sh"
echo 'KEY-IN-CHECKOUT' >"$tree/bench/manifest-go.yaml"
echo 'KEY-IN-CHECKOUT' >"$tree/.fixpoint/run/findings.json"
echo 'KEY-IN-HOME-FIXPOINT' >"$base/realfp/old.json"
echo 'CODEX-SESSION' >"$home/.codex/sessions/s.jsonl"
# codex_login <account> <refresh token> <last_refresh>: an auth.json as codex writes it
# (the fields the wrapper's sync reads).
codex_login() { printf '{"tokens":{"account_id":"%s","refresh_token":"%s"},"last_refresh":"%s"}\n' "$1" "$2" "$3"; }
codex_login acct-bench CODEX-AUTH 2026-11-01T00:00:00Z >"$home/.codex/auth.json"
echo 'model = "x"' >"$home/.codex/config.toml"
echo 'CLAUDE-TRANSCRIPT' >"$home/.claude/projects/p/t.jsonl"
echo 'CLAUDE-SESSION' >"$home/.claude/sessions/s.json"
echo 'CLAUDE-HISTORY' >"$home/.claude/history.jsonl"
mkdir -p "$home/.claude/debug" "$home/.claude/plans"
echo 'CLAUDE-DEBUG' >"$home/.claude/debug/d.txt"
echo 'CLAUDE-PLAN' >"$home/.claude/plans/p.md"
echo '{"hooks":{}}' >"$home/.claude/settings.json"
echo 'CLAUDE-MD' >"$home/.claude/CLAUDE.md"
echo '{"mcpServers":{}}' >"$home/.claude.json"

# The staged target, as run.sh makes it: a repo/ one level below a fresh host
# temp directory.
stage=$(mktemp -d)
stage=$(cd "$stage" && pwd -P)
target=$stage/repo
mkdir -p "$target"
echo 'TARGET-CONTENT' >"$target/a.txt"
trap 'rm -rf "$base" "$stage"' EXIT

# Host-side snapshots, to prove afterwards that nothing outside scratch changed.
snap() {
    (cd "$base" && find home tree realfp -type f 2>/dev/null | sort | while read -r f; do printf '%s %s\n' "$f" "$(cksum <"$f")"; done)
    (cd "$stage" && find . | sort)
    cat "$target/a.txt"
}
snap >"$base/before.snap"

# ---- the probe ----------------------------------------------------------------
# The agent's argv[0] names the CLI the wrapper provisions, so the probe is run as
# a stub named codex, claude or (anything else) other. It prints one line per
# attempt: <name>=ok when it worked, <name>=denied when not.
cat >"$home/.local/bin/probe.sh" <<'EOF'
#!/bin/sh
who=$1
# td is a destructive attempt: the unconfined control run must not perform it.
td() { [ -z "${CONTROL:-}" ] || return 0; t "$@"; }
t() {
    name=$1
    shift
    if "$@" >"$TMPDIR/o.$name" 2>&1; then echo "$name=ok"; else echo "$name=denied"; fi
    # What came back, whatever the exit status said: a sentinel in the output is a leak.
    if grep -q 'KEY-IN' "$TMPDIR/o.$name" 2>/dev/null; then echo "leak=$name"; fi
}
# A run that only has the wrapper sync the login, the agent doing nothing.
[ -z "${PROBE_QUIET:-}" ] || exit 0
echo "who=$who"
t cat_key cat "$REPO/bench/manifest-go.yaml"
t cat_glob sh -c 'cat "$REPO"/bench/manifest-*.yaml'
t cat_deep_glob sh -c 'cat "$REPO"/*/manifest-*.yaml'
# The earlier runs' findings, apart from the manifest: one cat of both would hide a
# successful read of either behind the other's failure.
t cat_findings cat "$REPO/.fixpoint/run/findings.json"
t cat_findings_glob sh -c 'cat "$REPO"/.fixpoint/*/*.json'
t find_key sh -c 'find "$REPO" -name "manifest-*" | grep .'
# A listing, not just ls: macOS refuses it, but a tmpfs mask on Linux is listable and
# EMPTY, so what must hold on both is that no entry of the real directory shows.
t ls_repo sh -c 'ls -A "$REPO" | grep .'
# The checkout through a process that stands in it (the test starts one, as
# fixpoint's own cwd is the checkout): reachable as /proc/<pid>/cwd unless the
# sandbox has its own pid namespace and proc.
t proc_any_cwd sh -c 'cat /proc/[0-9]*/cwd/bench/manifest-go.yaml 2>/dev/null | grep .'
t cat_fixpoint_home cat "$HOME/.fixpoint/old.json"
t ls_fixpoint_home sh -c 'ls -A "$HOME/.fixpoint" | grep .'
t cat_real_fixpoint sh -c 'cat "$REALFP"/* 2>/dev/null | grep .'
t read_target cat "$TARGET/a.txt"
td write_target sh -c 'echo x >"$TARGET/new.txt"'
td overwrite_target sh -c 'echo x >"$TARGET/a.txt"'
# The staged target's parent, writable on the host: renamed away, the target could
# be replaced by another tree at its own path.
td rename_parent sh -c 'mv "$PARENT" "$PARENT.moved"'
td replace_parent sh -c 'rm -rf "$PARENT" && mkdir -p "$PARENT/repo" && echo FAKE >"$PARENT/repo/a.txt"'
td write_parent sh -c 'echo x >"$PARENT/sibling"'
td write_home sh -c 'echo x >"$HOME/planted"'
t write_scratch sh -c 'echo x >"$TMPDIR/scratch-ok"'
t write_dev_null sh -c 'echo x >/dev/null'
td write_repo sh -c 'echo x >"$REPO/planted"'
# The live CLI state of BOTH CLIs, whichever one this is: planting a hook in
# either runs unconfined the next time the operator starts it.
t cat_codex_sessions cat "$HOME/.codex/sessions/s.jsonl"
t cat_codex_config cat "$HOME/.codex/config.toml"
td write_codex_config sh -c 'echo "[hooks]" >>"$HOME/.codex/config.toml"'
t cat_claude_transcript cat "$CLAUDE_CONFIG_DIR/projects/p/t.jsonl"
t cat_claude_session cat "$CLAUDE_CONFIG_DIR/sessions/s.json"
td write_claude_settings sh -c 'echo "{\"hooks\":{\"SessionStart\":[]}}" >"$CLAUDE_CONFIG_DIR/settings.json"'
td write_claude_md sh -c 'echo planted >"$CLAUDE_CONFIG_DIR/CLAUDE.md"'
td write_claude_plugins sh -c 'mkdir -p "$CLAUDE_CONFIG_DIR/plugins/marketplaces" && echo x >"$CLAUDE_CONFIG_DIR/plugins/marketplaces/x"'
td write_claude_commands sh -c 'mkdir "$CLAUDE_CONFIG_DIR/commands" && echo x >"$CLAUDE_CONFIG_DIR/commands/x.md"'
t read_claude_settings cat "$CLAUDE_CONFIG_DIR/settings.json"
# What a claude lens would leave for the operator's NEXT session: a planted memory
# file under projects/, a shell snapshot; and what it must not read: the history.
# (cat | grep ., not cat: on Linux the history is bound over by an EMPTY file.)
t cat_claude_debug cat "$CLAUDE_CONFIG_DIR/debug/d.txt"
t cat_claude_plans cat "$CLAUDE_CONFIG_DIR/plans/p.md"
t cat_claude_history sh -c 'cat "$CLAUDE_CONFIG_DIR/history.jsonl" 2>/dev/null | grep .'
td write_claude_projects sh -c 'mkdir -p "$CLAUDE_CONFIG_DIR/projects/-x/memory" && echo planted >"$CLAUDE_CONFIG_DIR/projects/-x/memory/MEMORY.md"'
td write_claude_snapshots sh -c 'echo planted >"$CLAUDE_CONFIG_DIR/shell-snapshots/snapshot-x.sh"'
# The claude CLI's shell tool directory is per invocation, not the one every lens on
# this target shares.
t claude_tmpdir_is_scratch sh -c '[ -n "$CLAUDE_CODE_TMPDIR" ] && [ "$CLAUDE_CODE_TMPDIR" = "$TMPDIR" ]'
# Another lens's scratch directory, which holds its findings (the test starts a lens
# that leaves a file there and passes its host path).
t read_other_scratch sh -c '[ -n "$SECRETPATH" ] && cat "$SECRETPATH"'
# What the CLI is given: the codex state is fresh and holds the credential.
t codex_home_is_fresh sh -c '[ -n "$CODEX_HOME" ] && [ "$CODEX_HOME" != "$HOME/.codex" ]'
t codex_home_auth cat "$CODEX_HOME/auth.json"
t cat_codex_auth cat "$HOME/.codex/auth.json"
t codex_home_auth_is_symlink sh -c '[ -L "$CODEX_HOME/auth.json" ]'
# A refreshed login saved the way codex 0.172 saves it: the link resolved, a temp file
# written beside the file it leads to, renamed over it.
td save_codex_auth sh -c 'f=$CODEX_HOME/auth.json; while [ -L "$f" ]; do l=$(readlink "$f"); case $l in /*) f=$l ;; *) f=${f%/*}/$l ;; esac; done; sed s/CODEX-AUTH/REFRESHED/ "$f" >"${f%/*}/.auth-probe" && mv -f "${f%/*}/.auth-probe" "$f"'
t codex_home_no_sessions sh -c '[ -z "$(ls "$CODEX_HOME/sessions" 2>/dev/null)" ]'
t codex_home_writable sh -c 'echo x >"$CODEX_HOME/new"'
# Sockets that start processes as the operator, outside any sandbox.
t user_bus sh -c 'ls "/run/user/$(id -u)/" 2>/dev/null | grep .'
t system_bus sh -c '[ -S /run/dbus/system_bus_socket ] || [ -S /var/run/docker.sock ]'
# The home directory as the CLI sees it: ~/.ssh and friends do not exist (Linux).
# run.sh puts bench/ on PATH so fixpoint can find the wrapper by name; the agent
# must not inherit an entry that names the checkout.
t path_clean sh -c 'case ":$PATH:" in *":$REPO/bench:"*) exit 1 ;; esac'
t see_ssh sh -c '[ -e "$HOME/.ssh" ]'
EOF
chmod +x "$home/.local/bin/probe.sh"
# Used by the pty probe (Linux): the controlling-terminal field of /proc/self/stat.
cat >"$home/.local/bin/ttyprobe" <<'EOF'
#!/bin/sh
echo "tty_nr=$(sed 's/.*) //' /proc/self/stat | cut -d' ' -f5)"
EOF
# A lens that leaves a file in its own scratch directory, for another to try to read.
cat >"$home/.local/bin/hold" <<'EOF'
#!/bin/sh
echo SECRET >"$TMPDIR/secret"
echo ready
while :; do sleep 1; done
EOF
# TIOCSTI: typing into the operator's terminal from a lens, through /dev/tty AND through
# the tty it inherited as stdin/stderr. Run under a pty by the test.
cat >"$home/.local/bin/stiprobe" <<'EOF'
#!/usr/bin/env python3
import fcntl, os, termios
def attempt(name, fd):
    try:
        fcntl.ioctl(fd, termios.TIOCSTI, b"x")
        print(name + "=injected")
    except OSError as e:
        print(name + "=failed")
try:
    attempt("devtty", os.open("/dev/tty", os.O_RDWR))
except OSError:
    print("devtty=failed")
attempt("stderr", 2)
attempt("stdin", 0)
EOF
chmod +x "$home/.local/bin/ttyprobe" "$home/.local/bin/hold" "$home/.local/bin/stiprobe"
for who in codex claude other; do
    cat >"$home/.local/bin/$who" <<EOF
#!/bin/sh
exec "$home/.local/bin/probe.sh" $who
EOF
    chmod +x "$home/.local/bin/$who"
done
# A planted .ssh on the fake home, for the Linux home-hiding probe.
mkdir -p "$home/.ssh"
echo 'KEY-SSH' >"$home/.ssh/id"

penv() { # the shared environment of every run of the probe, then the command
    env HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" REPO="$tree" REALFP="$base/realfp" TARGET="$target" PARENT="$stage" \
        PATH="$tree/bench:$home/.local/bin:$PATH" "$@"
}

wrap() { # <mode> <argv...>: the wrapper from the target, in the fake home
    m=$1
    shift
    (cd "$target" && penv "$tree/bench/sandbox.sh" "$target" "$m" -- "$@")
}

# ---- (a) refusals --------------------------------------------------------------
mkdir -p "$tree/bench/testdata/target-go"
# (the checkout's subtree as the target)
set +e
(cd "$tree" && penv "$tree/bench/sandbox.sh" "$tree/bench/testdata/target-go" ro -- /bin/echo RAN) >"$base/a1.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "a target inside the checkout was accepted"
! grep -q '^RAN$' "$base/a1.log" || fail "the agent ran for a target inside the checkout"
expect "the refusal names the problem" "$base/a1.log" 'inside'
mkdir -p "$home/.fixpoint/stage"
set +e
(cd "$home" && penv "$tree/bench/sandbox.sh" "$home/.fixpoint/stage" ro -- /bin/echo RAN) >"$base/a2.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] && ! grep -q '^RAN$' "$base/a2.log" || fail "a target inside ~/.fixpoint was accepted"
# A target that CONTAINS the checkout, rw: it would let the agent write what it
# cannot read, with the write allow landing after the checkout's deny.
set +e
(cd "$base" && penv "$tree/bench/sandbox.sh" "$base" rw -- /bin/echo RAN) >"$base/a3.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] && ! grep -q '^RAN$' "$base/a3.log" || fail "an rw target that contains the checkout was accepted"
expect "the refusal says the target contains it" "$base/a3.log" 'contains'
# ... and ro as well (nothing legitimate stages the target above the checkout).
set +e
(cd "$base" && penv "$tree/bench/sandbox.sh" "$base" ro -- /bin/echo RAN) >"$base/a4.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] && ! grep -q '^RAN$' "$base/a4.log" || fail "an ro target that contains the checkout was accepted"
# A sibling whose name shares the checkout's prefix is not inside it.
mkdir -p "$base/tree-other"
(cd "$base/tree-other" && penv "$tree/bench/sandbox.sh" "$base/tree-other" ro -- true) >"$base/a5.log" 2>&1 ||
    fail "a sibling directory sharing the checkout's name prefix was refused: $(cat "$base/a5.log")"
# Malformed invocations never reach the agent.
for bad in "ro" "sideways -- /bin/echo RAN" "ro /usr/bin/true /bin/echo RAN" "ro --"; do
    # shellcheck disable=SC2086
    if (cd "$target" && penv "$tree/bench/sandbox.sh" "$target" $bad) >"$base/b.log" 2>&1; then
        fail "malformed invocation '$bad' succeeded"
    fi
    ! grep -q '^RAN$' "$base/b.log" || fail "the agent ran for a malformed invocation '$bad'"
done
# A TMPDIR inside the checkout would put the scratch directory where it cannot be used.
set +e
(cd "$target" && penv TMPDIR="$tree" "$tree/bench/sandbox.sh" "$target" ro -- /bin/echo RAN) >"$base/a6.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] && ! grep -q '^RAN$' "$base/a6.log" || fail "a TMPDIR inside the checkout was accepted"

# refused <label> <cwd> <target> [NAME=value ...]: the wrapper must exit non-zero and
# the agent (/bin/echo RAN) must never run.
refused() {
    label=$1
    cdir=$2
    tgt=$3
    shift 3
    set +e
    (cd "$cdir" && penv "$@" "$tree/bench/sandbox.sh" "$tgt" ro -- /bin/echo RAN) >"$base/refused.log" 2>&1
    got=$?
    set -e
    if [ "$got" = 0 ] || grep -q '^RAN$' "$base/refused.log"; then
        fail "$label was accepted (exit $got)"
    elif ! grep -q -- "$REFUSAL" "$base/refused.log"; then
        # Refused, but by something else: on macOS a quote in the profile text makes
        # sandbox-exec itself fail, which would pass this test with plain() deleted.
        fail "$label was refused, but not by the wrapper's own check ($REFUSAL): $(head -c 200 "$base/refused.log")"
    fi
}

REFUSAL='TMPDIR'
# A TMPDIR that is the target or inside it would put the always-writable scratch
# directory inside what must stay read-only. `TMPDIR=.` from the agent's own
# directory IS the target.
mkdir -p "$target/sub" "$base/ok-tmp"
refused "a TMPDIR inside the target" "$target" "$target" TMPDIR="$target/sub"
refused "a TMPDIR that is the target" "$target" "$target" TMPDIR="$target"
refused "TMPDIR=. run from the target" "$target" "$target" TMPDIR=.
# ... while a TMPDIR beside it, and the default, are fine.
(cd "$target" && penv TMPDIR="$base/ok-tmp" "$tree/bench/sandbox.sh" "$target" ro -- /bin/echo RAN) >"$base/ok.log" 2>&1 ||
    fail "a TMPDIR beside the target was refused: $(cat "$base/ok.log")"
grep -q '^RAN$' "$base/ok.log" || fail "the agent did not run with a TMPDIR beside the target"

# plain(): a path the macOS profile would carry as TEXT must not contain a quote, a
# backslash or a newline, or it would change what the profile says. Every path in the
# profile goes through it, so each source of a path is tried: the target, the
# checkout, HOME, CLAUDE_CONFIG_DIR, TMPDIR (the scratch directory), and the symlinked
# ~/.fixpoint and ~/.codex that resolve to such a directory. The check runs before the
# OS split, so this holds on Linux as well.
REFUSAL='contains a quote, a backslash or a newline'
nlc=$(printf '\nx')
nlc=${nlc%x}
qd=$base/'we"ird'
bs=$base/'back\slash'
nld=$base/"new${nlc}line"
for d in "$qd" "$bs" "$nld"; do
    mkdir -p "$d"
    echo 'TARGET' >"$d/a.txt"
done
refused "a target whose name has a quote" "$qd" "$qd"
refused "a target whose name has a backslash" "$bs" "$bs"
refused "a target whose name has a newline" "$nld" "$nld"
refused "a HOME with a quote" "$target" "$target" HOME="$qd"
mkdir -p "$base/c\"laude" "$base/t\"mp"
refused "a CLAUDE_CONFIG_DIR with a quote" "$target" "$target" CLAUDE_CONFIG_DIR="$base/c\"laude"
refused "a TMPDIR with a quote" "$target" "$target" TMPDIR="$base/t\"mp"
# A refusal that comes AFTER the scratch directory exists (S itself has the quote)
# must not leave it behind.
[ "$(scratch_count "$base/t\"mp")" = 0 ] || fail "a refusal after mktemp left its scratch directory behind"
mkdir -p "$base/tr\"ee/bench"
cp "$tree/bench/sandbox.sh" "$base/tr\"ee/bench/sandbox.sh"
set +e
(cd "$target" && penv "$base/tr\"ee/bench/sandbox.sh" "$target" ro -- /bin/echo RAN) >"$base/refused.log" 2>&1
got=$?
set -e
{ [ "$got" != 0 ] && ! grep -q '^RAN$' "$base/refused.log" && grep -q -- "$REFUSAL" "$base/refused.log"; } || fail "a checkout whose path has a quote was not refused by plain(): $(head -c 200 "$base/refused.log")"
# ~/.fixpoint and ~/.codex as symlinks to such a directory: only their RESOLVED
# paths carry the quote.
for dot in .fixpoint .codex; do
    h=$base/home$dot
    mkdir -p "$h" "$base/ln\"$dot"
    ln -s "$base/ln\"$dot" "$h/$dot"
    refused "a $dot that resolves to a path with a quote" "$target" "$target" HOME="$h"
done

REFUSAL=''

# ---- (c) FAIL CLOSED ------------------------------------------------------------
# No sandbox tool on PATH, or an OS with none, and the agent does not run. PATH
# holds only what the wrapper itself needs; the agent is named by absolute path so
# that it WOULD run if the wrapper let it.
mkdir -p "$base/bin"
for tool in dirname uname; do
    ln -sf "$(command -v "$tool")" "$base/bin/$tool"
done
set +e
(cd "$target" && env HOME="$home" PATH="$base/bin" /bin/sh "$tree/bench/sandbox.sh" "$target" ro -- /usr/bin/touch "$base/ran-c") >"$base/c.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "a missing sandbox tool did not stop the wrapper"
[ ! -e "$base/ran-c" ] || fail "the agent ran with no sandbox tool available"
expect "the refusal says it will not run unconfined" "$base/c.log" 'unconfined'
rm -f "$base/bin/uname"
printf '#!/bin/sh\necho Plan9\n' >"$base/bin/uname"
chmod +x "$base/bin/uname"
set +e
(cd "$target" && env HOME="$home" PATH="$base/bin" /bin/sh "$tree/bench/sandbox.sh" "$target" ro -- /usr/bin/touch "$base/ran-c2") >"$base/c2.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "an unsupported OS did not stop the wrapper"
[ ! -e "$base/ran-c2" ] || fail "the agent ran on an unsupported OS"
expect "the refusal names the OS" "$base/c2.log" 'unsupported OS'

# ---- (c2) the wrapper's own plumbing, with a stub agent: status, signals, stdin ----
# The agent is a CHILD (so the scratch directory can be removed afterwards): its
# exit status and its stdin must reach it and come back, and a TERM must end it.
if [ "$os" = Darwin ] || [ "$os" = Linux ]; then
    ok_sandbox=0
    if [ "$os" = Darwin ]; then ok_sandbox=1; fi
    if [ "$os" = Linux ] && bwrap --unshare-pid --ro-bind / / true >/dev/null 2>&1; then ok_sandbox=1; fi
    if [ "$ok_sandbox" = 1 ]; then
        cat >"$home/.local/bin/plumb" <<'EOF'
#!/bin/sh
case $1 in
status) exit 7 ;;
stdin) cat >"$TMPDIR/stdin.out" && cat "$TMPDIR/stdin.out" ;;
term) trap 'echo got-term; exit 143' TERM; echo started; while :; do sleep 1; done ;;
term0) trap 'echo got-term; exit 0' TERM; echo started; while :; do sleep 1; done ;;
esac
EOF
        chmod +x "$home/.local/bin/plumb"
        set +e
        wrap ro plumb status >/dev/null 2>&1
        got=$?
        set -e
        [ "$got" = 7 ] || fail "the agent's exit status 7 came back as $got"
        out=$(printf 'PROMPT-TEXT' | (cd "$target" && penv "$tree/bench/sandbox.sh" "$target" ro -- plumb stdin))
        [ "$out" = PROMPT-TEXT ] || fail "stdin did not reach the agent (got '$out')"
        # TERM to the wrapper: it exits non-zero, the agent process is gone within a
        # few seconds, and S is removed. That is what holds on BOTH platforms. Whether
        # the agent also gets to run its own TERM handler differs: on macOS the wrapper
        # forwards it; on Linux bwrap does not forward signals and, with --die-with-parent
        # in a new pid namespace, kills the agent outright when it goes. fixpoint itself
        # only ever stops an agent with SIGKILL to the process group (internal/agent
        # KillProcessGroup), which no handler sees on either platform (not covered here).
        # The agent carries a token no other run on the host can have, and the run has a
        # TMPDIR of its own, so neither count can be skewed by a concurrent sandbox. The
        # second case is an agent that handles the TERM and exits 0: the wrapper must still
        # not report success for an interrupted run.
        token=term-$$-$(date +%s)
        mkdir -p "$base/term-tmp"
        for mode in term term0; do
            agent_running() { ps -eo args | grep -c "[p]lumb $mode $token" || true; }
            rm -f "$base/term.out"
            (cd "$target" && exec env HOME="$home" TMPDIR="$base/term-tmp" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$home/.local/bin:$PATH" "$tree/bench/sandbox.sh" "$target" ro -- plumb "$mode" "$token") >"$base/term.out" 2>&1 &
            wpid=$!
            n=0
            while ! grep -q started "$base/term.out" 2>/dev/null && [ "$n" -lt 100 ]; do
                sleep 0.1
                n=$((n + 1))
            done
            # Without this the rest could pass because the agent never ran at all.
            grep -q started "$base/term.out" || fail "$mode: the agent never started, so the TERM checks below mean nothing: $(cat "$base/term.out")"
            [ "$(agent_running)" -ge 1 ] || fail "$mode: the agent is not running before the TERM"
            [ "$(scratch_count "$base/term-tmp")" = 1 ] || fail "$mode: the wrapper has no scratch directory while the agent runs"
            kill -TERM "$wpid" 2>/dev/null || true
            reap "$wpid" 20
            [ "$reap_late" = 0 ] || fail "$mode: the wrapper did not exit within 20s of a TERM"
            [ "$reap_rc" != 0 ] || fail "$mode: a wrapper terminated by TERM exited 0"
            n=0
            while [ "$n" -lt 50 ] && { [ "$(agent_running)" -gt 0 ] || [ "$(scratch_count "$base/term-tmp")" -gt 0 ]; }; do
                sleep 0.1
                n=$((n + 1))
            done
            [ "$(agent_running)" = 0 ] || fail "$mode: the agent was still running 5s after its wrapper was terminated"
            [ "$(scratch_count "$base/term-tmp")" = 0 ] || fail "$mode: a terminated wrapper left its scratch directory behind"
            if [ "$os" = Darwin ]; then
                grep -q got-term "$base/term.out" || fail "$mode: TERM did not reach the agent"
            fi
        done
    else
        echo "bench/sandbox_test.sh: no working sandbox here; the status, stdin and signal checks were skipped." >&2
    fi
fi

# ---- (c3) the Linux branch's ARGUMENT ASSEMBLY, bwrap and uname stubbed ---------
# This proves the order of the mounts and that the agent's argv arrives intact
# behind them; it proves nothing about what bwrap then confines (see (d)).
mkdir -p "$base/lbin"
ln -sf "$(command -v dirname)" "$base/lbin/dirname"
printf '#!/bin/sh\necho Linux\n' >"$base/lbin/uname"
printf '#!/bin/sh\nfor a in "$@"; do printf "%%s\\n" "$a"; done\n' >"$base/lbin/bwrap"
chmod +x "$base/lbin/uname" "$base/lbin/bwrap"
mkdir -p "$home/.nvm"
for who in codex claude other; do
    for mode in ro rw; do
        out=$base/l-$who-$mode.out
        if ! (cd "$target" && env HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$base/lbin:$home/.local/bin:/usr/bin:/bin" \
            /bin/sh "$tree/bench/sandbox.sh" "$target" "$mode" -- "$who" --flag 'two words' -) >"$out" 2>"$out.err"; then
            fail "linux $who $mode: the wrapper failed: $(cat "$out.err")"
            continue
        fi
        [ "$(tail -4 "$out" | tr '\n' '|')" = "$who|--flag|two words|-|" ] || fail "linux $who $mode: the agent's argv did not arrive intact at the end"
        expect "linux $who $mode: new pid namespace" "$out" '^--unshare-pid$'
        expect "linux $who $mode: its own session, off the operator's terminal" "$out" '^--new-session$'
        expect "linux $who $mode: new ipc namespace" "$out" '^--unshare-ipc$'
        [ "$(grep -c -x /run "$out")" = 1 ] && [ "$(grep -x -B1 /run "$out" | head -1)" = --tmpfs ] ||
            fail "linux $who $mode: /run is not exactly one --tmpfs (the host's /run bound anywhere would show here twice)"
        if grep '^/run/' "$out" | grep -v -x '/run/systemd/resolve' | grep -q .; then
            fail "linux $who $mode: something under /run is bound beside the DNS directory"
        fi
        expect "linux $who $mode: no network namespace unshare (the CLI needs the API)" "$out" '^--unshare-(net|all)$' !
        expect "linux $who $mode: the home directory is replaced" "$out" "^$home\$"
        expect "linux $who $mode: ~/.fixpoint is masked" "$out" "^$base/realfp\$"
        expect "linux $who $mode: /tmp is replaced by the scratch directory" "$out" '^/tmp$'
        [ "$(grep -x -B1 '/tmp' "$out" | head -1 | sed 's|.*/||; s|\.[A-Za-z0-9]*$||')" = bench-sandbox ] || fail "linux $who $mode: /tmp is not the per-invocation scratch directory"
        # the checkout's mask comes after every bind that could expose it
        last_bind=$(grep -n -e '^--bind$' -e '^--ro-bind$' "$out" | tail -1 | cut -d: -f1)
        tmpfs_repo=$(grep -n -x -F "$tree" "$out" | tail -1 | cut -d: -f1)
        [ -n "$tmpfs_repo" ] && [ "$tmpfs_repo" -gt "$last_bind" ] || fail "linux $who $mode: the checkout is not masked after the last bind"
        # the target: ro-bound in ro, bound writable in rw, at its own path
        if [ "$mode" = ro ]; then
            [ "$(grep -x -B1 -F "$target" "$out" | head -1)" = "--ro-bind" ] || fail "linux $who ro: the target is not bound read-only"
        else
            [ "$(grep -x -B1 -F "$target" "$out" | head -1)" = "--bind" ] || fail "linux $who rw: the target is not bound writable"
        fi
        # nothing binds the real state directories wholesale
        expect "linux $who $mode: ~/.codex is not bound as a directory" "$out" "^$home/.codex\$" !
        case $who in
        codex)
            # no route file: no login is handed over
            expect "linux codex $mode: no login without the chatgpt route" "$out" "^$home/.codex/auth.json\$" !
            ;;
        claude)
            expect "linux claude $mode: transcripts are replaced by tmpfs" "$out" "^$home/.claude/projects\$"
            expect "linux claude $mode: the config dir is read-only" "$out" "^$home/.claude\$"
            [ "$(grep -x -B1 -F "$home/.claude" "$out" | head -1)" = "--ro-bind" ] || fail "linux claude $mode: the config dir is not bound read-only"
            [ "$(grep -x -B2 -F "$home/.claude/history.jsonl" "$out" | head -1)" = "--ro-bind" ] && [ "$(grep -x -B1 -F "$home/.claude/history.jsonl" "$out" | head -1)" = /dev/null ] ||
                fail "linux claude $mode: history.jsonl is not replaced by an empty file"
            ;;
        other) expect "linux other $mode: no CLI state at all" "$out" "^$home/\\.(claude|codex)" ! ;;
        esac
    done
done

# The chatgpt route binds the shared copy of the login, writable, and nothing else of
# ~/.codex (not the real auth.json: a rename cannot replace a bind-mounted file);
# a TMPDIR other than /tmp is masked, so another lens's scratch directory under it is
# not visible.
echo chatgpt >"$tree/bench/.sandbox-codex-auth"
mkdir -p "$base/ptmp"
(cd "$target" && env HOME="$home" TMPDIR="$base/ptmp" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$base/lbin:$home/.local/bin:/usr/bin:/bin" \
    /bin/sh "$tree/bench/sandbox.sh" "$target" ro -- codex x) >"$base/l-chatgpt.out" 2>"$base/l-chatgpt.err" || fail "linux chatgpt route: the wrapper failed: $(cat "$base/l-chatgpt.err")"
[ "$(grep -x -B1 -F "$home/.codex/.fixpoint-sandbox" "$base/l-chatgpt.out" | head -1)" = "--bind" ] || fail "linux chatgpt route: the shared login directory is not bound writable"
expect "linux chatgpt route: the real login is not bound" "$base/l-chatgpt.out" "^$home/.codex/auth.json\$" !
cmp -s "$home/.codex/auth.json" "$home/.codex/.fixpoint-sandbox/auth.json" || fail "linux chatgpt route: the shared copy does not match the real login"
rm -rf "$home/.codex/.fixpoint-sandbox" "$home/.codex/auth.json.lock"
expect "linux: a TMPDIR other than /tmp is masked" "$base/l-chatgpt.out" "^$(cd "$base/ptmp" && pwd -P)\$"
rm -f "$tree/bench/.sandbox-codex-auth"

# ---- (d) the real boundary --------------------------------------------------------
can_confine=0
case $os in
Darwin) command -v sandbox-exec >/dev/null 2>&1 && can_confine=1 ;;
Linux)
    if command -v bwrap >/dev/null 2>&1 && bwrap --unshare-pid --unshare-ipc --ro-bind / / --proc /proc --dev /dev true >/dev/null 2>&1; then
        can_confine=1
    fi
    ;;
esac
if [ "$can_confine" = 1 ]; then
    # A process whose cwd is the checkout, as fixpoint's is: on Linux its
    # /proc/<pid>/cwd is a way to the checkout unless the sandbox has its own pid
    # namespace and /proc.
    (cd "$tree" && exec sleep 300) &
    decoy=$!
    trap 'rc=$?; { kill "$decoy" || true; wait "$decoy" || true; } 2>/dev/null; rm -rf "$base" "$stage"; exit "$rc"' EXIT
    # Positive control: unconfined, every sentinel is there to be read.
    : >"$base/control.out"
    (cd "$target" && penv CONTROL=1 TMPDIR="$base" "$home/.local/bin/probe.sh" control) >"$base/control.out" 2>&1 || true
    if [ "$os" = Linux ]; then
        expect "control: a decoy's /proc/<pid>/cwd reaches the checkout unconfined" "$base/control.out" '^proc_any_cwd=ok$'
    fi
    for k in cat_findings cat_findings_glob cat_deep_glob cat_claude_debug cat_claude_plans cat_codex_auth cat_claude_history cat_key cat_glob find_key ls_repo ls_fixpoint_home cat_fixpoint_home cat_real_fixpoint read_target cat_codex_sessions cat_codex_config cat_claude_transcript cat_claude_session read_claude_settings; do
        expect "control: $k works unconfined, so its denial below means something" "$base/control.out" "^$k=ok\$"
    done
    # the key sentinels really do come back unconfined: a missing "leak=" below means something
    expect "control: the manifest's sentinel does come back unconfined" "$base/control.out" '^leak=cat_key$'
    expect "control: the findings' sentinel does come back unconfined" "$base/control.out" '^leak=cat_findings$'
    grep -q '^user_bus=ok$' "$base/control.out" || echo "bench/sandbox_test.sh: no user bus socket on this host, so 'the user bus is not exposed' was not checked." >&2
    grep -q '^system_bus=ok$' "$base/control.out" || echo "bench/sandbox_test.sh: no system bus or docker socket on this host, so 'the system sockets are not exposed' was not checked." >&2
    snap >"$base/before.snap"

    # run_probe <who> <mode> <outfile>
    run_probe() {
        (cd "$target" && penv TMPDIR="$base/ptmp" "$tree/bench/sandbox.sh" "$target" "$2" -- "$home/.local/bin/$1") >"$3" 2>"$3.err" ||
            fail "the probe could not run as $1 in $2 mode: $(cat "$3.err")"
    }
    for who in codex claude other; do
        mkdir -p "$base/ptmp"
        scratch_before=$(scratch_count "$base/ptmp")
        run_probe "$who" ro "$base/$who.out"
        o=$base/$who.out
        expect "$who: the checkout's manifest is unreadable" "$o" '^cat_key=denied$'
        expect "$who: a glob cannot reach the manifest" "$o" '^cat_glob=denied$'
        expect "$who: a deeper glob cannot reach the manifest" "$o" '^cat_deep_glob=denied$'
        expect "$who: the earlier findings are unreadable" "$o" '^cat_findings=denied$'
        expect "$who: a glob cannot reach the earlier findings" "$o" '^cat_findings_glob=denied$'
        expect "$who: no sentinel of the key or the findings came back in any output" "$o" '^leak=' !
        expect "$who: find does not see the manifest" "$o" '^find_key=denied$'
        expect "$who: the checkout lists no entries" "$o" '^ls_repo=denied$'
        expect "$who: no /proc/<pid>/cwd leads to the checkout" "$o" '^proc_any_cwd=denied$'
        expect "$who: ~/.fixpoint is unreadable" "$o" '^cat_fixpoint_home=denied$'
        expect "$who: ~/.fixpoint lists no entries" "$o" '^ls_fixpoint_home=denied$'
        expect "$who: a symlinked ~/.fixpoint hides its real directory" "$o" '^cat_real_fixpoint=denied$'
        expect "$who: the agent's PATH does not name the checkout" "$o" '^path_clean=ok$'
        expect "$who: the target is readable" "$o" '^read_target=ok$'
        expect "$who: scratch can be written" "$o" '^write_scratch=ok$'
        expect "$who: /dev/null can be written" "$o" '^write_dev_null=ok$'
        expect "$who: the codex sessions are unreadable" "$o" '^cat_codex_sessions=denied$'
        expect "$who: codex's config.toml is unreadable" "$o" '^cat_codex_config=denied$'
        # Only meaningful when the same socket is there to see unconfined; a host with no
        # user bus or system socket proves nothing, and says so once below.
        if grep -q '^user_bus=ok$' "$base/control.out"; then
            expect "$who: the user bus is not exposed" "$o" '^user_bus=denied$'
        fi
        if grep -q '^system_bus=ok$' "$base/control.out"; then
            expect "$who: the system sockets are not exposed" "$o" '^system_bus=denied$'
        fi
        if [ "$os" = Darwin ]; then
            # macOS denies the write; Linux gives a tmpfs that vanishes, which the
            # host-side comparison below covers for both.
            expect "$who: the target cannot get a new file" "$o" '^write_target=denied$'
            expect "$who: the target cannot be overwritten" "$o" '^overwrite_target=denied$'
            expect "$who: the target's parent cannot be renamed" "$o" '^rename_parent=denied$'
            expect "$who: the target's parent cannot be replaced" "$o" '^replace_parent=denied$'
            expect "$who: the target's parent cannot be written" "$o" '^write_parent=denied$'
            expect "$who: HOME cannot be written" "$o" '^write_home=denied$'
            expect "$who: the checkout cannot be written" "$o" '^write_repo=denied$'
        else
            expect "$who: ~/.ssh does not exist inside" "$o" '^see_ssh=denied$'
        fi
        # The host, afterwards: nothing outside scratch has changed, whatever the
        # sandbox let the agent believe it did.
        snap >"$base/after.snap"
        if ! cmp -s "$base/before.snap" "$base/after.snap"; then
            fail "$who: the host changed under a read-only lens:"
            diff "$base/before.snap" "$base/after.snap" | sed 's/^/    /' || true
        fi
        # S is removed, whichever way the agent ended
        [ "$(scratch_count "$base/ptmp")" = "$scratch_before" ] || fail "$who: a scratch directory was left behind"
        # a repaired world for the next persona (Linux binds do not change the host,
        # but a mac failure above could have)
        snap >"$base/before.snap"
    done
    # what each CLI IS given
    expect "codex: its CODEX_HOME is a fresh directory" "$base/codex.out" '^codex_home_is_fresh=ok$'
    # No route file: this is a route that carries its own key, so there is no login in
    # the fresh home and the live one is unreadable, even though it exists.
    expect "codex: no ChatGPT login without the chatgpt route" "$base/codex.out" '^codex_home_auth=denied$'
    for who in codex claude other; do
        expect "$who: the live codex login is unreadable without the chatgpt route" "$base/$who.out" '^cat_codex_auth=denied$'
        expect "$who: claude's prompt history is unreadable" "$base/$who.out" '^cat_claude_history=denied$'
        expect "$who: claude's debug logs are unreadable" "$base/$who.out" '^cat_claude_debug=denied$'
        expect "$who: claude's plans are unreadable" "$base/$who.out" '^cat_claude_plans=denied$'
    done
    expect "claude: its shell-tool directory is its own scratch" "$base/claude.out" '^claude_tmpdir_is_scratch=ok$'
    expect "codex: it starts with no sessions" "$base/codex.out" '^codex_home_no_sessions=ok$'
    expect "codex: its fresh state is writable" "$base/codex.out" '^codex_home_writable=ok$'
    if [ "$os" = Darwin ]; then
        expect "codex: it cannot write the live config.toml" "$base/codex.out" '^write_codex_config=denied$'
    fi
    expect "claude: the transcripts are unreadable" "$base/claude.out" '^cat_claude_transcript=denied$'
    expect "claude: its other sessions are unreadable" "$base/claude.out" '^cat_claude_session=denied$'
    expect "claude: settings.json cannot be modified (a SessionStart hook)" "$base/claude.out" '^write_claude_settings=denied$'
    expect "claude: CLAUDE.md cannot be modified" "$base/claude.out" '^write_claude_md=denied$'
    expect "claude: commands/ cannot be added" "$base/claude.out" '^write_claude_commands=denied$'
    expect "claude: plugins/ (hooks, skills, MCP servers) cannot be written" "$base/claude.out" '^write_claude_plugins=denied$'
    if [ "$os" = Darwin ]; then
        # A memory file planted under projects/ is read by the operator's next session.
        expect "claude: the live projects/ cannot be written" "$base/claude.out" '^write_claude_projects=denied$'
        expect "claude: shell-snapshots/ cannot be written" "$base/claude.out" '^write_claude_snapshots=denied$'
    fi
    expect "claude: settings.json can still be read" "$base/claude.out" '^read_claude_settings=ok$'
    for who in codex other; do
        expect "$who: the live claude transcripts are unreadable" "$base/$who.out" '^cat_claude_transcript=denied$'
        expect "$who: the live claude settings are unreadable" "$base/$who.out" '^read_claude_settings=denied$'
    done
    expect "claude: the live codex config is unreadable" "$base/claude.out" '^cat_codex_config=denied$'

    # The chatgpt route: only then is the login handed over (to a codex lens, and not to
    # any other binary), through a symlink in the fresh CODEX_HOME to the one file.
    echo chatgpt >"$tree/bench/.sandbox-codex-auth"
    run_probe codex ro "$base/codex-chatgpt.out"
    # The entry in the fresh home is a symlink to the shared copy, and a refresh saved
    # through it the way codex saves one (temp file and rename) reaches the host's login
    # when the wrapper exits.
    shared=$home/.codex/.fixpoint-sandbox/auth.json
    expect "codex: the fresh home's login is a symlink" "$base/codex-chatgpt.out" '^codex_home_auth_is_symlink=ok$'
    expect "codex: a refreshed login can be saved as codex saves it" "$base/codex-chatgpt.out" '^save_codex_auth=ok$'
    grep -q REFRESHED "$home/.codex/auth.json" || fail "codex: the refreshed login did not reach the host's auth.json"
    cmp -s "$home/.codex/auth.json" "$shared" || fail "codex: the shared copy and the host's login differ after the run"
    quiet_codex() { (cd "$target" && penv PROBE_QUIET=1 TMPDIR="$base/ptmp" "$tree/bench/sandbox.sh" "$target" ro -- "$home/.local/bin/codex") >"$base/quiet.out" 2>&1 || fail "the quiet codex run failed: $(cat "$base/quiet.out")"; }
    # A refresh a wrapper killed outright (a group SIGKILL on a timeout) left in the
    # shared copy reaches the host at the next start.
    codex_login acct-bench CODEX-AUTH 2026-11-01T00:00:00Z >"$home/.codex/auth.json"
    mkdir -p "${shared%/*}"
    codex_login acct-bench KILLED 2026-11-02T00:00:00Z >"$shared"
    quiet_codex
    grep -q '"KILLED"' "$home/.codex/auth.json" || fail "codex: a refresh a killed wrapper left behind did not reach the host's login"
    # A login refreshed outside the bench replaces an older shared copy.
    codex_login acct-bench OUTSIDE 2026-11-03T00:00:00Z >"$home/.codex/auth.json"
    quiet_codex
    grep -q '"OUTSIDE"' "$shared" || fail "codex: a login refreshed outside did not replace the shared copy"
    # A newer login for ANOTHER account in the shared copy is not taken.
    codex_login acct-other FOREIGN 2026-11-04T00:00:00Z >"$shared"
    quiet_codex
    grep -q '"OUTSIDE"' "$home/.codex/auth.json" || fail "codex: a login for another account replaced the host's"
    cmp -s "$home/.codex/auth.json" "$shared" || fail "codex: the shared copy was not reset after a login for another account"
    [ -z "$(ls -A "$home/.codex" | grep '^\.fixpoint-sandbox-auth\.')" ] || fail "codex: the login sync left temp files in ~/.codex"
    rm -rf "$home/.codex/.fixpoint-sandbox" "$home/.codex/auth.json.lock"
    codex_login acct-bench CODEX-AUTH 2026-11-01T00:00:00Z >"$home/.codex/auth.json"
    run_probe other ro "$base/other-chatgpt.out"
    rm -f "$tree/bench/.sandbox-codex-auth"
    # A route file that is not EXACTLY "chatgpt" and one newline hands nothing over: a
    # word that is not chatgpt, a second line, a blank line, trailing junk, no newline,
    # a carriage return.
    n=0
    for content in 'none\n' 'chatgpt\nsomething\n' 'chatgpt\n\n' 'chatgpt junk\n' 'chatgpt' 'chatgpt\r\n'; do
        n=$((n + 1))
        printf "$content" >"$tree/bench/.sandbox-codex-auth"
        run_probe codex ro "$base/codex-marker$n.out"
        expect "codex: a route file holding '$content' hands over no login" "$base/codex-marker$n.out" '^codex_home_auth=denied$'
        expect "codex: ... and the live login stays unreadable ('$content')" "$base/codex-marker$n.out" '^cat_codex_auth=denied$'
    done
    rm -f "$tree/bench/.sandbox-codex-auth"
    grep -q '^codex_home_auth_is_symlink=ok$' "$base/codex-marker2.out" && fail "codex: a malformed route file left a login symlink in the fresh home"
    expect "codex: the chatgpt route gets the login through its fresh home" "$base/codex-chatgpt.out" '^codex_home_auth=ok$'
    expect "codex: ... and nothing else of the live ~/.codex" "$base/codex-chatgpt.out" '^cat_codex_config=denied$'
    expect "codex: ... whose sessions stay unreadable" "$base/codex-chatgpt.out" '^cat_codex_sessions=denied$'
    expect "another binary is not given the login even on the chatgpt route" "$base/other-chatgpt.out" '^cat_codex_auth=denied$'
    snap >"$base/after.snap"
    cmp -s "$base/before.snap" "$base/after.snap" || fail "the chatgpt route changed the host"

    # Parallel lenses: another lens's scratch directory (its findings) is not readable.
    mkdir -p "$base/ptmp"
    (cd "$target" && exec env HOME="$home" TMPDIR="$base/ptmp" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$home/.local/bin:$PATH" "$tree/bench/sandbox.sh" "$target" ro -- hold) >"$base/hold.out" 2>&1 &
    holdpid=$!
    n=0
    while ! grep -q ready "$base/hold.out" 2>/dev/null && [ "$n" -lt 100 ]; do
        sleep 0.1
        n=$((n + 1))
    done
    grep -q ready "$base/hold.out" || fail "the first lens never started: $(cat "$base/hold.out")"
    holdscratch=$(ls -d "$base"/ptmp/bench-sandbox.* 2>/dev/null | head -1)
    [ -n "$holdscratch" ] && [ "$(cat "$holdscratch/secret" 2>/dev/null)" = SECRET ] || fail "the first lens left no file in its scratch directory to try to read (control)"
    (cd "$target" && penv TMPDIR="$base/ptmp" SECRETPATH="$holdscratch/secret" "$tree/bench/sandbox.sh" "$target" ro -- "$home/.local/bin/other") >"$base/second.out" 2>"$base/second.err" || fail "the second lens could not run: $(cat "$base/second.err")"
    expect "a second lens cannot read the first one's scratch directory" "$base/second.out" '^read_other_scratch=denied$'
    kill -TERM "$holdpid" 2>/dev/null || true
    reap "$holdpid" 20

    # TIOCSTI: typing into the operator's terminal. A lens inherits it unless it has a
    # session of its own. Linux gets one (bwrap --new-session); macOS does not (setsid
    # would leak agents past fixpoint's group kill, and macOS has no die-with-parent), so
    # the profile denies the ioctl on the terminal devices instead. Either way, under a pty
    # the confined probe must FAIL on /dev/tty and on its inherited tty fds, while the
    # unconfined control SUCCEEDS (else the probe proves nothing). python3 makes the pty.
    if command -v python3 >/dev/null 2>&1; then
        ptyrun() { python3 -c 'import pty,sys; sys.exit(pty.spawn(sys.argv[1:]) >> 8)' "$@" 2>&1 | tr -d '\r'; }
        ptyrun "$home/.local/bin/stiprobe" >"$base/sti-control.out" || true
        if grep -q 'devtty=injected$' "$base/sti-control.out" && grep -q 'stderr=injected$' "$base/sti-control.out"; then
            (cd "$target" && ptyrun env HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$home/.local/bin:$PATH" "$tree/bench/sandbox.sh" "$target" ro -- stiprobe) >"$base/sti.out" || true
            expect "the lens cannot type into /dev/tty (TIOCSTI)" "$base/sti.out" 'devtty=failed$'
            expect "the lens cannot type into its inherited stderr tty (TIOCSTI)" "$base/sti.out" 'stderr=failed$'
            expect "the lens cannot type into its inherited stdin tty (TIOCSTI)" "$base/sti.out" 'stdin=failed$'
            expect "nothing was injected at all" "$base/sti.out" 'injected' !
        else
            echo "bench/sandbox_test.sh: could not make TIOCSTI work unconfined under a pty here; the TIOCSTI check was skipped." >&2
        fi
        if [ "$os" = Linux ]; then
            ptyrun "$home/.local/bin/ttyprobe" >"$base/tty-control.out" || true
            if grep -q '^tty_nr=[1-9]' "$base/tty-control.out"; then
                (cd "$target" && ptyrun env HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$home/.local/bin:$PATH" "$tree/bench/sandbox.sh" "$target" ro -- ttyprobe) >"$base/tty.out" || true
                expect "linux: the lens has no controlling terminal" "$base/tty.out" '^tty_nr=0$'
            else
                echo "bench/sandbox_test.sh: could not give a probe a controlling terminal here; the new-session check was skipped." >&2
            fi
        fi
    fi

    # rw: the target is the only extra place to write; nothing else opens.
    wr=$(mktemp -d)
    wr=$(cd "$wr" && pwd -P)
    echo 'X' >"$wr/a.txt"
    (cd "$wr" && penv TARGET="$wr" "$tree/bench/sandbox.sh" "$wr" rw -- "$home/.local/bin/other") >"$base/rw.out" 2>"$base/rw.err" ||
        fail "the probe could not run in rw mode: $(cat "$base/rw.err")"
    expect "rw: the target can get a new file" "$base/rw.out" '^write_target=ok$'
    expect "rw: the checkout is still unreadable" "$base/rw.out" '^cat_key=denied$'
    [ -f "$wr/new.txt" ] || fail "rw: the write to the target did not land on the host"
    [ ! -e "$home/planted" ] || fail "rw: HOME was written"
    rm -rf "$wr"
else
    echo "bench/sandbox_test.sh: no working sandbox here ($os${1:+}); the confinement probes were SKIPPED and prove nothing. The refusals, fail-closed cases and the bwrap argument assembly above still ran." >&2
fi

if [ "$fails" -ne 0 ]; then
    echo "bench/sandbox_test.sh: $fails failure(s)" >&2
    exit 1
fi
echo "bench/sandbox_test.sh: refusals, fail-closed cases, argument assembly and (where a sandbox works) confinement OK"
