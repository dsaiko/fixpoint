#!/bin/sh
# Filesystem boundary for a bench lens: run an agent CLI where it cannot read
# the answer key and cannot write outside what the CLI itself needs.
#
#   bench/sandbox.sh <target-dir> <ro|rw> -- <agent argv...>
#
# It is the `sandbox.command` of config/bench-code.yaml and
# config/bench-design.yaml, which fixpoint calls as
#   bench/sandbox.sh {{target}} {{target_mode}} -- <the agent's own argv>
# and it execs that argv inside the sandbox. There is no unconfined path out of
# this script: a missing sandbox tool, an unsupported OS or a bad argument is a
# non-zero exit before the agent starts.
#
# WHAT IT ENFORCES
#   - READ: the repository checkout (the directory holding bench/, which holds
#     the manifests, the scorer, every results row and its own .fixpoint/) and
#     the user's ~/.fixpoint (old bench logs, other models' findings) do not
#     exist for the agent. This is a boundary, unlike score.py's key_reads(),
#     which only reads the lens's tool calls afterwards and cannot see a path
#     the shell forms by expansion (`cat /*/*/fix*/b*/m*`) or a claude-harness
#     route at all.
#   - WRITE: nothing but the target (rw mode only), temp directories, /dev/null
#     and the agent CLI's own state. A reviewer in `ro` mode cannot modify the
#     tree it reviews or leave anything in $HOME.
#
# WHAT IT DOES NOT
#   - The agent can still READ the rest of $HOME (~/.ssh, ~/.aws, the CLI's
#     own credentials) and use the network. Confining those would break the
#     CLIs, and the bench's concern is the answer key, not a hostile model.
#   - Sandboxes do not nest, on either platform. An agent that starts its own
#     sandbox for its shell commands (codex `--sandbox read-only` is seatbelt
#     on macOS, bubblewrap on Linux) cannot do it in here, so the bench
#     candidate's own sandbox is switched off and THIS script is what enforces
#     read-only; bench/run.sh does that in the generated candidate file only.
#     fixpoint refuses that file's can_edit: false with --sandbox
#     danger-full-access unless the invocation says -agents-sandboxed (a flag,
#     never a config key; docs/security.md), which bench/run.sh passes.
#
# PLATFORMS
#   macOS  sandbox-exec, an `allow default` profile with file-read* denied on
#          the two hidden paths and file-write* denied everywhere then allowed
#          on a short list. Tested: bench/sandbox_test.sh runs real probes
#          through it, and a codex and a claude session ran inside it.
#   Linux  bwrap, a read-only view of / with the writable paths bound over it
#          and the hidden paths replaced by empty tmpfs. NOT TESTED: written
#          from bwrap's documented behaviour and one measured fact (`--tmpfs
#          <repo>` hides the checkout on the bench host). Read it as a first
#          draft until a real run on that host has been checked.
#   other  refused.
set -eu

die() {
    echo "bench/sandbox.sh: $*" >&2
    exit 2
}

[ $# -ge 4 ] || die "usage: bench/sandbox.sh <target-dir> <ro|rw> -- <agent argv...>"
target=$1
mode=$2
shift 2
[ "$1" = -- ] || die "usage: bench/sandbox.sh <target-dir> <ro|rw> -- <agent argv...>"
shift
[ $# -ge 1 ] || die "no agent command after --"
case $mode in
ro | rw) ;;
*) die "mode '$mode' is not ro or rw" ;;
esac

# Physical paths: seatbelt matches the resolved path of what a process opens
# (/var is a symlink to /private/var on macOS), so a profile written with the
# symlinked spelling would not match anything.
[ -d "$target" ] || die "target '$target' is not a directory"
target=$(cd "$target" && pwd -P) || die "cannot resolve target '$target'"
repo=$(cd "$(dirname "$0")/.." && pwd -P) || die "cannot resolve the checkout"
[ -n "${HOME:-}" ] && [ -d "$HOME" ] || die "HOME is not set to a directory"
home=$(cd "$HOME" && pwd -P) || die "cannot resolve HOME"

# Hiding the checkout hides a target inside it, and the agent would review an
# empty directory and be scored for it. ~/.fixpoint is hidden too, and for the
# same reason may not hold the target.
for hidden in "$repo" "$home/.fixpoint"; do
    case "$target/" in
    "$hidden"/*) die "target '$target' is inside '$hidden', which this wrapper hides from the agent; stage the target outside it (bench/run.sh does)" ;;
    esac
done

# fixpoint finds this script by NAME, through a PATH that run.sh extends with
# this directory (a relative wrapper path would resolve against the target, not
# the checkout; see config/bench-code.yaml). The agent does not need that entry
# and it would hand it the checkout's path in its own environment, so it goes.
here=$repo/bench
newpath=
oldifs=$IFS
IFS=:
set -f
for d in ${PATH:-}; do
    [ "$d" = "$here" ] || newpath="${newpath:+$newpath:}$d"
done
set +f
IFS=$oldifs
PATH=$newpath
export PATH

# Scratch directories the agent may write: TMPDIR when it is set, plus the
# system ones. Resolved, and only those that exist.
scratch=
for d in "${TMPDIR:-}" /tmp /var/tmp; do
    [ -n "$d" ] && [ -d "$d" ] || continue
    d=$(cd "$d" && pwd -P) || continue
    scratch="$scratch $d"
done

case $(uname -s) in
Darwin)
    command -v sandbox-exec >/dev/null 2>&1 || die "sandbox-exec not found; refusing to run the agent unconfined"
    # Paths travel as -D parameters and are read with (param ...), never spliced
    # into the profile text, so a quote or backslash in a path cannot change
    # what the profile says.
    #
    # SBPL is last-match-wins, which is why the order below is load-bearing:
    # the blanket write deny comes first and the allows after it. The two hidden
    # paths and the target are then settled LAST, because TMPDIR, which allows
    # writes, may hold any of them (the staged target always does).
    #
    # Writable, each with its reason:
    #   /dev/null /dev/tty /dev/dtracehelper  shells and runtimes open them
    #   TMPDIR /private/tmp /private/var/tmp  scratch; the node and Go runtimes
    #                                         and every shell command use it
    #   ~/.codex                              codex's sessions, sqlite state
    #                                         and logs; it will not start
    #                                         without being able to write them
    #   $CLAUDE_CONFIG_DIR, else ~/.claude    the claude CLI's state (fixpoint
    #   and ~/.claude.json                    passes CLAUDE_CONFIG_DIR through,
    #                                         and with it set the CLI keeps
    #                                         .claude.json inside that directory)
    #   the target, only in rw mode           the one place an editing agent
    #                                         is meant to write
    profile='(version 1)
(allow default)
(deny file-read* (subpath (param "REPO")) (subpath (param "FPDIR")))
(deny file-write*)
(allow file-write*
    (literal "/dev/null") (literal "/dev/tty") (literal "/dev/dtracehelper")
    (subpath (param "TMP")) (subpath "/private/tmp") (subpath "/private/var/tmp")
    (subpath (param "CODEX")) (subpath (param "CLAUDE")) (literal (param "CLAUDEJSON")))
(deny file-write* (subpath (param "REPO")) (subpath (param "FPDIR")))'
    if [ "$mode" = rw ]; then
        profile="$profile
(allow file-write* (subpath (param \"TARGET\")))"
    else
        profile="$profile
(deny file-write* (subpath (param \"TARGET\")))"
    fi
    tmp=$(cd "${TMPDIR:-/private/tmp}" 2>/dev/null && pwd -P) || tmp=/private/tmp
    claude=$home/.claude
    [ -n "${CLAUDE_CONFIG_DIR:-}" ] && [ -d "$CLAUDE_CONFIG_DIR" ] && claude=$(cd "$CLAUDE_CONFIG_DIR" && pwd -P)
    exec sandbox-exec \
        -D "REPO=$repo" -D "FPDIR=$home/.fixpoint" -D "TARGET=$target" -D "TMP=$tmp" \
        -D "CODEX=$home/.codex" -D "CLAUDE=$claude" -D "CLAUDEJSON=$home/.claude.json" \
        -p "$profile" "$@"
    ;;
Linux)
    command -v bwrap >/dev/null 2>&1 || die "bwrap not found; refusing to run the agent unconfined"
    # POSIX sh has no arrays, so the mount arguments are appended after the
    # agent's argv in "$@" and the agent's argv is then rotated behind them.
    # Order matters, later mounts sit on top of earlier ones:
    #   --ro-bind / /      the whole filesystem, readable and not writable
    #   --bind <path>      what may be written: scratch, the CLI's state, and
    #                      the target in rw mode; each only if it exists
    #   --ro-bind target   puts a ro target back to read-only when a scratch
    #                      directory above it was bound writable
    #   --tmpfs <hidden>   last, so nothing above can re-expose it
    argc=$#
    claude=$home/.claude
    [ -n "${CLAUDE_CONFIG_DIR:-}" ] && [ -d "$CLAUDE_CONFIG_DIR" ] && claude=$(cd "$CLAUDE_CONFIG_DIR" && pwd -P)
    for d in $scratch "$home/.codex" "$claude"; do
        if [ -d "$d" ]; then set -- "$@" --bind "$d" "$d"; fi
    done
    if [ -f "$home/.claude.json" ]; then set -- "$@" --bind "$home/.claude.json" "$home/.claude.json"; fi
    if [ "$mode" = rw ]; then
        set -- "$@" --bind "$target" "$target"
    else
        set -- "$@" --ro-bind "$target" "$target"
    fi
    set -- "$@" --tmpfs "$repo"
    if [ -d "$home/.fixpoint" ]; then set -- "$@" --tmpfs "$home/.fixpoint"; fi
    n=0
    while [ "$n" -lt "$argc" ]; do
        a=$1
        shift
        set -- "$@" "$a"
        n=$((n + 1))
    done
    # --dev /dev and --proc /proc: a fresh /dev (null, tty, urandom) and the
    # /proc the CLIs read. --die-with-parent ties the agent to fixpoint's
    # lifetime, so the process-group kill still reaches it through bwrap.
    exec bwrap --die-with-parent --ro-bind / / --dev /dev --proc /proc "$@"
    ;;
*)
    die "unsupported OS '$(uname -s)'; refusing to run the agent unconfined"
    ;;
esac
