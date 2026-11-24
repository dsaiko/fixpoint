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

# expect <label> <file> <grep -E pattern> [!]: the file must (or, with !, must
# not) match.
expect() {
    if grep -Eq -- "$3" "$2"; then hit=1; else hit=0; fi
    if [ "${4:-}" = "!" ]; then hit=$((1 - hit)); fi
    [ "$hit" = 1 ] || fail "$1"
}

# scratch_count: how many wrapper scratch directories the host temp directory holds
# now; a run must leave it as it found it, whatever else is lying about.
scratch_count() { ls "${TMPDIR:-/tmp}" 2>/dev/null | grep -c '^bench-sandbox\.' || true; }

# ---- the stand-in world -------------------------------------------------------
tree=$base/tree
home=$base/home
mkdir -p "$tree/bench" "$tree/.fixpoint/run" "$home/.local/bin" "$base/realfp"
# ~/.fixpoint is a SYMLINK to a directory elsewhere: the profile must hide the real one.
ln -s "$base/realfp" "$home/.fixpoint"
mkdir -p "$home/.codex/sessions" "$home/.claude/projects/p" "$home/.claude/sessions"
cp "$SANDBOX_SH" "$tree/bench/sandbox.sh"
chmod +x "$tree/bench/sandbox.sh"
echo 'KEY-IN-CHECKOUT' >"$tree/bench/manifest-go.yaml"
echo 'KEY-IN-CHECKOUT' >"$tree/.fixpoint/run/findings.json"
echo 'KEY-IN-HOME-FIXPOINT' >"$base/realfp/old.json"
echo 'CODEX-SESSION' >"$home/.codex/sessions/s.jsonl"
echo 'CODEX-AUTH' >"$home/.codex/auth.json"
echo 'model = "x"' >"$home/.codex/config.toml"
echo 'CLAUDE-TRANSCRIPT' >"$home/.claude/projects/p/t.jsonl"
echo 'CLAUDE-SESSION' >"$home/.claude/sessions/s.json"
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
t() { name=$1; shift; if "$@" >"$TMPDIR/o.$name" 2>&1; then echo "$name=ok"; else echo "$name=denied"; fi; }
echo "who=$who"
t cat_key cat "$REPO/bench/manifest-go.yaml"
t cat_glob sh -c 'cat "$REPO"/bench/manifest-*.yaml'
t cat_deep_glob sh -c 'cat "$REPO"/*/manifest-*.yaml "$REPO"/.fixpoint/*/*.json'
t find_key sh -c 'find "$REPO" -name "manifest-*" | grep .'
t ls_repo ls "$REPO"
# The checkout through a process that stands in it (the test starts one, as
# fixpoint's own cwd is the checkout): reachable as /proc/<pid>/cwd unless the
# sandbox has its own pid namespace and proc.
t proc_any_cwd sh -c 'cat /proc/[0-9]*/cwd/bench/manifest-go.yaml 2>/dev/null | grep .'
t cat_fixpoint_home cat "$HOME/.fixpoint/old.json"
t ls_fixpoint_home ls "$HOME/.fixpoint"
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
td write_claude_commands sh -c 'mkdir "$CLAUDE_CONFIG_DIR/commands" && echo x >"$CLAUDE_CONFIG_DIR/commands/x.md"'
t read_claude_settings cat "$CLAUDE_CONFIG_DIR/settings.json"
# What the CLI is given: the codex state is fresh and holds the credential.
t codex_home_is_fresh sh -c '[ -n "$CODEX_HOME" ] && [ "$CODEX_HOME" != "$HOME/.codex" ]'
t codex_home_auth cat "$CODEX_HOME/auth.json"
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
# exit status, its stdin and a TERM must still reach it and come back.
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
        # TERM to the wrapper reaches the agent, the wrapper exits, and S is gone.
        before=$(scratch_count)
        (cd "$target" && exec env HOME="$home" CLAUDE_CONFIG_DIR="$home/.claude" PATH="$home/.local/bin:$PATH" "$tree/bench/sandbox.sh" "$target" ro -- plumb term) >"$base/term.out" 2>&1 &
        wpid=$!
        n=0
        while ! grep -q started "$base/term.out" 2>/dev/null && [ "$n" -lt 100 ]; do
            sleep 0.1
            n=$((n + 1))
        done
        kill -TERM "$wpid" 2>/dev/null || true
        wait "$wpid" 2>/dev/null || true
        after=$(scratch_count)
        [ "$after" = "$before" ] || fail "a terminated wrapper left its scratch directory behind"
        grep -q got-term "$base/term.out" || fail "TERM did not reach the agent"
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
        expect "linux $who $mode: new ipc namespace" "$out" '^--unshare-ipc$'
        expect "linux $who $mode: /run is masked" "$out" '^/run$'
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
        codex) expect "linux codex $mode: only the credential file is bound" "$out" "^$home/.codex/auth.json\$" ;;
        claude)
            expect "linux claude $mode: transcripts are replaced by tmpfs" "$out" "^$home/.claude/projects\$"
            expect "linux claude $mode: the config dir is read-only" "$out" "^$home/.claude\$"
            [ "$(grep -x -B1 -F "$home/.claude" "$out" | head -1)" = "--ro-bind" ] || fail "linux claude $mode: the config dir is not bound read-only"
            ;;
        other) expect "linux other $mode: no CLI state at all" "$out" "^$home/\\.(claude|codex)" ! ;;
        esac
    done
done

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
    for k in cat_key cat_glob find_key cat_fixpoint_home cat_real_fixpoint read_target cat_codex_sessions cat_codex_config cat_claude_transcript cat_claude_session read_claude_settings; do
        expect "control: $k works unconfined, so its denial below means something" "$base/control.out" "^$k=ok\$"
    done
    snap >"$base/before.snap"

    # run_probe <who> <mode> <outfile>
    run_probe() {
        (cd "$target" && penv "$tree/bench/sandbox.sh" "$target" "$2" -- "$home/.local/bin/$1") >"$3" 2>"$3.err" ||
            fail "the probe could not run as $1 in $2 mode: $(cat "$3.err")"
    }
    for who in codex claude other; do
        scratch_before=$(scratch_count)
        run_probe "$who" ro "$base/$who.out"
        o=$base/$who.out
        expect "$who: the checkout's manifest is unreadable" "$o" '^cat_key=denied$'
        expect "$who: a glob cannot reach the manifest" "$o" '^cat_glob=denied$'
        expect "$who: a deeper glob cannot reach the manifest or the findings" "$o" '^cat_deep_glob=denied$'
        expect "$who: find does not see the manifest" "$o" '^find_key=denied$'
        expect "$who: the checkout cannot be listed" "$o" '^ls_repo=denied$'
        expect "$who: no /proc/<pid>/cwd leads to the checkout" "$o" '^proc_any_cwd=denied$'
        expect "$who: ~/.fixpoint is unreadable" "$o" '^cat_fixpoint_home=denied$'
        expect "$who: ~/.fixpoint cannot be listed" "$o" '^ls_fixpoint_home=denied$'
        expect "$who: a symlinked ~/.fixpoint hides its real directory" "$o" '^cat_real_fixpoint=denied$'
        expect "$who: the agent's PATH does not name the checkout" "$o" '^path_clean=ok$'
        expect "$who: the target is readable" "$o" '^read_target=ok$'
        expect "$who: scratch can be written" "$o" '^write_scratch=ok$'
        expect "$who: /dev/null can be written" "$o" '^write_dev_null=ok$'
        expect "$who: the codex sessions are unreadable" "$o" '^cat_codex_sessions=denied$'
        expect "$who: codex's config.toml is unreadable" "$o" '^cat_codex_config=denied$'
        expect "$who: the user bus is not exposed" "$o" '^user_bus=denied$'
        expect "$who: the system sockets are not exposed" "$o" '^system_bus=denied$'
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
        [ "$(scratch_count)" = "$scratch_before" ] || fail "$who: a scratch directory was left behind"
        # a repaired world for the next persona (Linux binds do not change the host,
        # but a mac failure above could have)
        snap >"$base/before.snap"
    done
    # what each CLI IS given
    expect "codex: its CODEX_HOME is a fresh directory" "$base/codex.out" '^codex_home_is_fresh=ok$'
    expect "codex: its credential is reachable through it" "$base/codex.out" '^codex_home_auth=ok$'
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
    expect "claude: settings.json can still be read" "$base/claude.out" '^read_claude_settings=ok$'
    for who in codex other; do
        expect "$who: the live claude transcripts are unreadable" "$base/$who.out" '^cat_claude_transcript=denied$'
        expect "$who: the live claude settings are unreadable" "$base/$who.out" '^read_claude_settings=denied$'
    done
    expect "claude: the live codex config is unreadable" "$base/claude.out" '^cat_codex_config=denied$'

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
