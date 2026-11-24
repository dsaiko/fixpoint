#!/bin/sh
# Tests for bench/sandbox.sh, the filesystem boundary around a bench lens. Run by
# `make bench-check`.
#
# The wrapper is copied into a scratch tree that stands in for the checkout, so
# "the checkout" is a directory this test owns and plants an answer key in. The
# refusals and the fail-closed cases run everywhere; the real confinement runs
# only where the wrapper has been tested, which is macOS (sandbox-exec). The
# Linux branch (bwrap) is NOT exercised here and is not claimed to work.
#
# SANDBOX_SH=<path> tests another copy of the wrapper: that is how each assertion
# was mutation-checked (break the wrapper, see this test fail, restore).
set -eu

here=$(cd "$(dirname "$0")" && pwd)
SANDBOX_SH=${SANDBOX_SH:-$here/sandbox.sh}
tmp=$(mktemp -d)
tmp=$(cd "$tmp" && pwd -P)
trap 'rm -rf "$tmp"' EXIT

# The stand-in checkout, with a planted key, and a stand-in home with a .fixpoint.
# The checkout sits in TMPDIR, which the wrapper makes writable, so that only its
# own final deny keeps the checkout unwritable (a real one under /tmp would be).
tree=$tmp/scratch/tree
# The target sits in TMPDIR, as run.sh stages it: the allow list for scratch
# covers it, so only the wrapper's own final deny keeps a ro target read-only.
target=$tmp/scratch/target
mkdir -p "$tree/bench" "$tree/.fixpoint/run" "$tmp/home/.fixpoint" "$target"
cp "$SANDBOX_SH" "$tree/bench/sandbox.sh"
chmod +x "$tree/bench/sandbox.sh"
echo 'KEY-IN-CHECKOUT' >"$tree/bench/manifest-go.yaml"
echo 'KEY-IN-CHECKOUT' >"$tree/.fixpoint/run/findings.json"
echo 'KEY-IN-HOME-FIXPOINT' >"$tmp/home/.fixpoint/old.json"
echo 'TARGET-CONTENT' >"$target/a.txt"

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

wrap() { # <target> <mode> <argv...>, in the scratch HOME and TMPDIR, cwd = target
    t=$1
    shift
    (cd "$t" && env HOME="$tmp/home" TMPDIR="$tmp/scratch" PATH="$PATH" "$tree/bench/sandbox.sh" "$t" "$@")
}

# (a) A target inside the checkout is refused: hiding the checkout would hide
# it, and the agent would review an empty directory.
mkdir -p "$tree/bench/testdata/target-go"
set +e
wrap "$tree/bench/testdata/target-go" ro -- touch "$tmp/scratch/ran-a" >"$tmp/a.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "a target inside the checkout was accepted"
[ ! -e "$tmp/scratch/ran-a" ] || fail "the agent ran for a target inside the checkout"
expect "the refusal names the problem" "$tmp/a.log" 'inside'
# ... and so is one inside ~/.fixpoint, which is hidden as well.
mkdir -p "$tmp/home/.fixpoint/stage"
set +e
wrap "$tmp/home/.fixpoint/stage" ro -- touch "$tmp/scratch/ran-a2" >"$tmp/a2.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "a target inside ~/.fixpoint was accepted"
[ ! -e "$tmp/scratch/ran-a2" ] || fail "the agent ran for a target inside ~/.fixpoint"
# A target merely NAMED like the checkout's prefix is not inside it.
mkdir -p "$tmp/scratch/tree-other"
if [ "$(uname -s)" = Darwin ]; then
    wrap "$tmp/scratch/tree-other" ro -- true >"$tmp/a3.log" 2>&1 || fail "a sibling directory sharing the checkout's name prefix was refused: $(cat "$tmp/a3.log")"
fi

# (b) Malformed invocations never reach the agent.
for bad in "ro" "sideways -- touch $tmp/scratch/ran-b" "ro /usr/bin/true /usr/bin/touch $tmp/scratch/ran-b" "ro --"; do
    # shellcheck disable=SC2086
    if wrap "$target" $bad >"$tmp/b.log" 2>&1; then
        fail "malformed invocation '$bad' succeeded"
    fi
done
[ ! -e "$tmp/scratch/ran-b" ] || fail "the agent ran for a malformed invocation"

# (c) FAIL CLOSED: no sandbox tool on PATH, or an OS with none, and the agent
# does not run. PATH holds only what the wrapper itself needs, so neither
# sandbox-exec nor bwrap can be found; the agent is named by absolute path so
# that it WOULD run if the wrapper let it.
mkdir -p "$tmp/bin"
for tool in dirname uname; do
    ln -sf "$(command -v "$tool")" "$tmp/bin/$tool"
done
set +e
(cd "$target" && env HOME="$tmp/home" TMPDIR="$tmp/scratch" PATH="$tmp/bin" /bin/sh "$tree/bench/sandbox.sh" "$target" ro -- /usr/bin/touch "$tmp/scratch/ran-c") >"$tmp/c.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "a missing sandbox tool did not stop the wrapper"
[ ! -e "$tmp/scratch/ran-c" ] || fail "the agent ran with no sandbox tool available"
expect "the refusal says it will not run unconfined" "$tmp/c.log" 'unconfined'
# An OS the wrapper does not know.
rm -f "$tmp/bin/uname"
printf '#!/bin/sh\necho Plan9\n' >"$tmp/bin/uname"
chmod +x "$tmp/bin/uname"
set +e
(cd "$target" && env HOME="$tmp/home" TMPDIR="$tmp/scratch" PATH="$tmp/bin" /bin/sh "$tree/bench/sandbox.sh" "$target" ro -- /usr/bin/touch "$tmp/scratch/ran-c2") >"$tmp/c2.log" 2>&1
got=$?
set -e
[ "$got" != 0 ] || fail "an unsupported OS did not stop the wrapper"
[ ! -e "$tmp/scratch/ran-c2" ] || fail "the agent ran on an unsupported OS"
expect "the refusal names the OS" "$tmp/c2.log" 'unsupported OS'

# (c2) The Linux branch's ARGUMENT ASSEMBLY, with bwrap and uname stubbed: this
# proves the order of the mounts and that the agent's argv arrives intact behind
# them, not that bwrap confines anything (that is untested; see the header).
mkdir -p "$tmp/lbin" "$tmp/home/.codex"
for tool in dirname; do
    ln -sf "$(command -v "$tool")" "$tmp/lbin/$tool"
done
printf '#!/bin/sh\necho Linux\n' >"$tmp/lbin/uname"
printf '#!/bin/sh\nfor a in "$@"; do printf "%%s\\n" "$a"; done\n' >"$tmp/lbin/bwrap"
chmod +x "$tmp/lbin/uname" "$tmp/lbin/bwrap"
for mode in ro rw; do
    if ! (cd "$target" && env HOME="$tmp/home" TMPDIR="$tmp/scratch" PATH="$tmp/lbin" /bin/sh "$tree/bench/sandbox.sh" "$target" "$mode" -- agent-cli --flag 'two words' -) >"$tmp/l-$mode.out" 2>"$tmp/l-$mode.err"; then
        fail "linux $mode: the wrapper failed: $(cat "$tmp/l-$mode.err")"
        continue
    fi
    # The agent's argv is the tail, whole.
    [ "$(tail -4 "$tmp/l-$mode.out" | tr '\n' '|')" = "agent-cli|--flag|two words|-|" ] || fail "linux $mode: the agent's argv did not arrive intact at the end"
    expect "linux $mode: the root is read-only" "$tmp/l-$mode.out" '^--ro-bind$'
    # The checkout is hidden by a tmpfs that comes after every bind that could expose it.
    last_bind=$(grep -n -e '^--bind$' -e '^--ro-bind$' "$tmp/l-$mode.out" | tail -1 | cut -d: -f1)
    tmpfs_repo=$(grep -n -x -F "$tree" "$tmp/l-$mode.out" | tail -1 | cut -d: -f1)
    [ -n "$tmpfs_repo" ] && [ "$tmpfs_repo" -gt "$last_bind" ] || fail "linux $mode: the checkout is not hidden after the last bind"
    expect "linux $mode: ~/.fixpoint is hidden" "$tmp/l-$mode.out" "^$tmp/home/.fixpoint\$"
done
# ro puts the target back to read-only, rw binds it writable: the flag before the
# target's path is the one that governs it.
[ "$(grep -x -B1 -F "$target" "$tmp/l-ro.out" | head -1)" = "--ro-bind" ] || fail "linux ro: the target is not bound read-only"
[ "$(grep -x -B1 -F "$target" "$tmp/l-rw.out" | head -1)" = "--bind" ] || fail "linux rw: the target is not bound writable"

# (d) The real boundary, where it has been tested.
if [ "$(uname -s)" = Darwin ] && command -v sandbox-exec >/dev/null 2>&1; then
    probe=$tmp/probe.sh
    cat >"$probe" <<'EOF'
#!/bin/sh
# Prints one line per thing tried: <name>=ok when it worked, <name>=denied when not.
# Each attempt's own output goes to $OUT.<name>, in scratch, so a positive result
# proves the redirect and the command both worked rather than that nothing ran.
t() { name=$1; shift; if "$@" >"$OUT.$name" 2>&1; then echo "$name=ok"; else echo "$name=denied"; fi; }
t cat_key cat "$REPO/bench/manifest-go.yaml"
# The glob must not rescue it: the shell can neither list the directory nor open
# what it names. Left unexpanded the pattern is a literal, which cat cannot open.
t cat_glob sh -c 'cat "$REPO"/bench/manifest-*.yaml'
t cat_deep_glob sh -c 'cat "$REPO"/*/manifest-*.yaml "$REPO"/.fixpoint/*/*.json'
t find_key sh -c 'find "$REPO" -name "manifest-*" | grep .'
t ls_repo ls "$REPO"
t cat_fixpoint_home cat "$HOME/.fixpoint/old.json"
t ls_fixpoint_home ls "$HOME/.fixpoint"
t read_target cat a.txt
t write_target sh -c 'echo x >new.txt'
t overwrite_target sh -c 'echo x >a.txt'
t write_home sh -c 'echo x >"$HOME/planted"'
t write_scratch sh -c 'echo x >"$TMPDIR/scratch-ok"'
t write_dev_null sh -c 'echo x >/dev/null'
t write_repo sh -c 'echo x >"$REPO/planted"'
# run.sh puts bench/ on PATH so fixpoint can find the wrapper by name; the agent
# must not inherit an entry that names the checkout.
t path_clean sh -c 'case ":$PATH:" in *":$REPO/bench:"*) exit 1 ;; esac'
EOF
    chmod +x "$probe"
    run_probe() { # <mode> <outfile>
        rm -f "$target/new.txt" "$tmp/home/planted" "$tmp/scratch/scratch-ok"
        (cd "$target" && env HOME="$tmp/home" TMPDIR="$tmp/scratch" PATH="$tree/bench:$PATH" REPO="$tree" OUT="$tmp/scratch/o" \
            "$tree/bench/sandbox.sh" "$target" "$1" -- "$probe") >"$2" 2>"$2.err" ||
            fail "the probe could not run in $1 mode: $(cat "$2.err")"
    }
    run_probe ro "$tmp/ro.out"
    expect "ro: the checkout's manifest is unreadable" "$tmp/ro.out" '^cat_key=denied$'
    expect "ro: a glob cannot reach the manifest" "$tmp/ro.out" '^cat_glob=denied$'
    expect "ro: a deeper glob cannot reach the manifest or the findings" "$tmp/ro.out" '^cat_deep_glob=denied$'
    expect "ro: find does not see the manifest" "$tmp/ro.out" '^find_key=denied$'
    expect "ro: the checkout cannot be listed" "$tmp/ro.out" '^ls_repo=denied$'
    expect "ro: ~/.fixpoint is unreadable" "$tmp/ro.out" '^cat_fixpoint_home=denied$'
    expect "ro: ~/.fixpoint cannot be listed" "$tmp/ro.out" '^ls_fixpoint_home=denied$'
    expect "ro: the target is readable" "$tmp/ro.out" '^read_target=ok$'
    expect "ro: the target cannot get a new file" "$tmp/ro.out" '^write_target=denied$'
    expect "ro: the target cannot be overwritten" "$tmp/ro.out" '^overwrite_target=denied$'
    expect "ro: HOME cannot be written" "$tmp/ro.out" '^write_home=denied$'
    expect "ro: scratch can be written" "$tmp/ro.out" '^write_scratch=ok$'
    expect "ro: /dev/null can be written" "$tmp/ro.out" '^write_dev_null=ok$'
    expect "ro: the checkout cannot be written" "$tmp/ro.out" '^write_repo=denied$'
    expect "ro: the agent's PATH does not name the checkout" "$tmp/ro.out" '^path_clean=ok$'
    # Nothing the probe printed may carry the key's content.
    cat "$tmp"/scratch/o.* >"$tmp/all.out" 2>/dev/null || true
    expect "no probe output carries the key" "$tmp/all.out" 'KEY-IN' !
    [ ! -e "$tmp/home/planted" ] && [ ! -e "$target/new.txt" ] && [ "$(cat "$target/a.txt")" = TARGET-CONTENT ] ||
        fail "ro: something was written to the target or HOME"
    rm -f "$tmp"/scratch/o.*

    # rw is the one mode in which the target may be written; nothing else opens.
    run_probe rw "$tmp/rw.out"
    expect "rw: the target can get a new file" "$tmp/rw.out" '^write_target=ok$'
    expect "rw: the checkout is still unreadable" "$tmp/rw.out" '^cat_key=denied$'
    expect "rw: HOME is still not writable" "$tmp/rw.out" '^write_home=denied$'
    rm -f "$tmp"/scratch/o.* "$target/new.txt"
    echo 'TARGET-CONTENT' >"$target/a.txt"
else
    echo "bench/sandbox_test.sh: no sandbox-exec here ($(uname -s)); the confinement checks are macOS-only and were skipped. The bwrap branch of bench/sandbox.sh is untested." >&2
fi

if [ "$fails" -ne 0 ]; then
    echo "bench/sandbox_test.sh: $fails failure(s)" >&2
    exit 1
fi
echo "bench/sandbox_test.sh: sandbox.sh refusals and fail-closed cases OK"
