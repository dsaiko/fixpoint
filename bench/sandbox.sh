#!/bin/sh
# Filesystem boundary for a bench lens: run an agent CLI where it cannot read
# the answer key and has only the state it needs.
#
#   bench/sandbox.sh <target-dir> <ro|rw> -- <agent argv...>
#
# It is the `sandbox.command` of config/bench-code.yaml and
# config/bench-design.yaml, which fixpoint calls as
#   sandbox.sh {{target}} {{target_mode}} -- <the agent's own argv>
# (by NAME: bench/run.sh puts bench/ on PATH; a relative path would resolve
# against the staged target). There is no unconfined path out of this script: a
# missing sandbox tool, an unsupported OS or a bad argument is a non-zero exit
# before the agent starts. The agent runs as a child, not by exec, so that its
# scratch state is removed afterwards; its exit status and stdin pass through.
#
# SIGNALS. TERM, INT and HUP to the wrapper are forwarded to the sandbox tool, the
# wrapper exits non-zero and removes S. On macOS the agent then receives the TERM and
# may run its own handler. On Linux it does not: bwrap does not forward signals and,
# with --die-with-parent in a new pid namespace, kills the agent outright when it
# goes, so the agent is simply gone. That is also all fixpoint ever does to an agent:
# on a timeout or cancel it SIGKILLs the whole process group (KillProcessGroup in
# internal/agent), a signal no handler sees on either platform, and the wrapper's
# scratch directory is then left in TMPDIR (it holds no credential, see below).
#
# THE PRINCIPLE is to expose what a lens needs and nothing else, so the policy
# is built from what is granted, not from what is denied:
#
#   - The repository checkout (the directory holding bench/: manifests, scorer,
#     results, its own .fixpoint/) and ~/.fixpoint (earlier runs' findings) are
#     hidden. score.py's key_reads() only reads tool calls afterwards; this is the
#     boundary it cannot be (a path the shell forms by expansion, a claude-harness
#     route whose raw log has no tool calls).
#   - The lens writes only a per-invocation scratch directory S (mktemp under
#     TMPDIR, mode 700, removed on exit) and, in `rw` mode, the target.
#   - CLI state is DISPOSABLE. The CLI is taken from argv[0] (codex, claude; any
#     other binary gets no state at all):
#       codex   a fresh CODEX_HOME inside S (on Linux that is under /tmp, and codex
#               prints "could not create PATH aliases: Refusing to create helper
#               binaries under temporary dir" and carries on; harmless so far). The only thing put in it is a SYMLINK
#               to the real ~/.codex/auth.json when there is one (ChatGPT auth;
#               the ollama-direct@ and codex@ routes authenticate by env key and
#               need nothing). The symlink, not a copy, so no credential is ever
#               written to scratch and a killed wrapper leaves none behind. The
#               rest of the real ~/.codex (sessions, sqlite, config.toml) is
#               unreadable and unwritable.
#       claude  the REAL config directory, with the transcripts unreadable and
#               everything that is executable configuration read-only. A fresh
#               CLAUDE_CONFIG_DIR does not work on macOS: the login lives in the
#               Keychain under a name derived from the directory's path, and a
#               fresh path is "Not logged in" (measured 2026-11-24). Copying the
#               credential out would also risk a refresh-token rotation that logs
#               the operator out. So: settings.json, CLAUDE.md, commands/,
#               skills/, plugins/ (plugin hooks, skills, commands and MCP servers) and hooks
#               cannot be modified; projects/, sessions/,
#               file-history/ and the other transcript stores cannot be read (on
#               macOS they are write-only, on Linux an empty tmpfs). What stays
#               WRITABLE is .claude.json, which is global state that includes
#               user-scope MCP servers: a lens that writes one there plants code
#               that runs the next time the operator starts claude. That cannot
#               be closed while the claude CLI needs to rewrite that file.
#
# WHAT IT DOES NOT ENFORCE (read this before trusting a score)
#   - The agent can still read most of $HOME on macOS (~/.ssh, ~/.aws, other
#     repositories, including other CHECKOUTS of this one, which hold the same
#     manifests), the CLI's own credential, and use the network.
#   - On Linux the home directory is hidden except the toolchain directories the
#     CLI is installed in, but the network namespace is shared with the host:
#     abstract unix sockets (an X or D-Bus abstract address) and localhost
#     services remain reachable. /run, including /run/user/<uid> and the system
#     sockets under it, is masked.
#   - Sandboxes do not nest, on either platform. An agent that starts its own
#     sandbox for its shell commands (codex `--sandbox read-only` is seatbelt on
#     macOS, bubblewrap on Linux) cannot in here, so the bench candidate's own
#     sandbox is switched off and THIS script is what enforces read-only;
#     bench/run.sh does that in the generated candidate file only, and passes
#     -agents-sandboxed, the flag that lets fixpoint accept it (docs/security.md).
#   - The local-ollama routes (`ollama launch ...`) need ~/.ollama state this
#     wrapper does not provide; bench/run.sh refuses them.
#
# PLATFORMS
#   macOS  sandbox-exec, an `allow default` profile: reads denied on the hidden
#          paths, writes denied everywhere then allowed on S (and rw target, and
#          the claude state paths). Tested by bench/sandbox_test.sh and by real
#          codex and claude sessions.
#   Linux  bwrap: the host root read-only, /run masked, the home directory
#          replaced by an empty tmpfs with only the toolchain and CLI state bound
#          back, /tmp replaced by S, the target bound read-only at its own path,
#          the checkout and ~/.fixpoint masked by empty tmpfs, own PID and IPC
#          namespaces (so /proc/<pid>/cwd of fixpoint is not reachable). The
#          network is shared: the CLI needs the API. Tested on the bench host by
#          the operator; bench/sandbox_test.sh runs a real probe where bwrap works.
#   other  refused.
set -eu

nl='
'

die() {
    echo "bench/sandbox.sh: $*" >&2
    exit 2
}

usage="usage: bench/sandbox.sh <target-dir> <ro|rw> -- <agent argv...>"
[ $# -ge 4 ] || die "$usage"
target=$1
mode=$2
shift 2
[ "$1" = -- ] || die "$usage"
shift
[ $# -ge 1 ] || die "no agent command after --"
case $mode in
ro | rw) ;;
*) die "mode '$mode' is not ro or rw" ;;
esac

# plain <path> refuses a path the macOS profile could not carry verbatim. The
# profile is text; a quote, a backslash or a newline in a path would change what
# it says, so such a path is refused rather than escaped.
plain() {
    case $1 in
    *\"* | *\\* | *"$nl"*) die "path '$1' contains a quote, a backslash or a newline" ;;
    esac
}

# phys <dir> prints the physical path of an existing directory, or nothing.
phys() { (cd "$1" 2>/dev/null && pwd -P) || true; }

# Physical paths: seatbelt matches the resolved path of what a process opens
# (/var is a symlink to /private/var on macOS) and a hidden path that is a
# symlink would hide only the link.
[ -d "$target" ] || die "target '$target' is not a directory"
target=$(phys "$target")
[ -n "$target" ] || die "cannot resolve the target"
repo=$(phys "$(dirname "$0")/..")
[ -n "$repo" ] || die "cannot resolve the checkout"
[ -n "${HOME:-}" ] && [ -d "$HOME" ] || die "HOME is not set to a directory"
home=$(phys "$HOME")
[ -n "$home" ] || die "cannot resolve HOME"
fpdir=$home/.fixpoint
if [ -e "$fpdir" ]; then
    # Resolved: a symlinked ~/.fixpoint would otherwise leave its real directory
    # outside the profile's read deny.
    fpdir=$(phys "$fpdir")
    [ -n "$fpdir" ] || die "$home/.fixpoint exists but is not a directory I can resolve"
fi
codexdir=$home/.codex
[ ! -e "$codexdir" ] || codexdir=$(phys "$codexdir")
[ -n "$codexdir" ] || die "$home/.codex exists but is not a directory I can resolve"
claudedir=${CLAUDE_CONFIG_DIR:-$home/.claude}
[ ! -e "$claudedir" ] || claudedir=$(phys "$claudedir")
[ -n "$claudedir" ] || die "the claude config dir exists but is not a directory I can resolve"
if [ -n "${CLAUDE_CONFIG_DIR:-}" ]; then
    claudejson=$claudedir/.claude.json
else
    claudejson=$home/.claude.json
fi
for p in "$target" "$repo" "$home" "$fpdir" "$codexdir" "$claudedir" "$claudejson"; do plain "$p"; done

# Hiding the checkout hides a target inside it, and the agent would review an
# empty directory and be scored for it. A writable target that CONTAINS a hidden
# path would let the agent modify what it cannot read, so that is refused too.
for hidden in "$repo" "$fpdir"; do
    case "$target/" in
    "$hidden"/*) die "target '$target' is inside '$hidden', which this wrapper hides from the agent; stage the target outside it (bench/run.sh does)" ;;
    esac
    case "$hidden/" in
    "$target"/*) die "target '$target' contains '$hidden', which this wrapper hides from the agent; stage the target outside it (bench/run.sh does)" ;;
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

# Which CLI, from argv[0]. An unknown binary is given no state at all.
cli=${1##*/}
case $cli in
codex | claude) ;;
*) cli=other ;;
esac
agent_path=$(command -v "$1" 2>/dev/null) || agent_path=

os=$(uname -s)
case $os in
Darwin) command -v sandbox-exec >/dev/null 2>&1 || die "sandbox-exec not found; refusing to run the agent unconfined" ;;
Linux) command -v bwrap >/dev/null 2>&1 || die "bwrap not found; refusing to run the agent unconfined" ;;
*) die "unsupported OS '$os'; refusing to run the agent unconfined" ;;
esac

# The per-invocation scratch directory S. A TMPDIR inside a hidden path would
# put it where the agent cannot use it.
tb=$(phys "${TMPDIR:-/tmp}")
[ -n "$tb" ] || die "TMPDIR '${TMPDIR:-/tmp}' is not a directory"
for hidden in "$repo" "$fpdir"; do
    case "$tb/" in
    "$hidden"/*) die "TMPDIR '$tb' is inside '$hidden', which this wrapper hides from the agent" ;;
    esac
done
# ... and a TMPDIR at or inside the TARGET (TMPDIR=. from the agent's directory is
# the target) would put the always-writable scratch inside what must stay read-only.
case "$tb/" in
"$target"/*) die "TMPDIR '$tb' is the target or inside it '$target'; the scratch directory must not be part of what is reviewed" ;;
esac
S=$(mktemp -d "$tb/bench-sandbox.XXXXXX") || die "cannot create a scratch directory in '$tb'"
S=$(phys "$S")
[ -n "$S" ] || die "cannot resolve the scratch directory"
plain "$S"
chmod 700 "$S"

child=
cleanup() { rm -rf "$S"; }
# A signal is forwarded to the agent and the wait below resumes; EXIT removes S
# whichever way the wrapper ends. (A SIGKILL cannot be trapped, which is why no
# credential is ever copied into S: see the codex note above.)
on_signal() {
    [ -z "$child" ] || kill -TERM "$child" 2>/dev/null || true
}
trap cleanup EXIT
trap on_signal TERM INT HUP

# Where S is, from inside. On Linux it IS the sandbox's /tmp.
case $os in
Linux) inner=/tmp ;;
*) inner=$S ;;
esac

# Disposable CLI state.
TMPDIR=$inner
export TMPDIR
case $cli in
codex)
    mkdir "$S/codex"
    if [ -f "$codexdir/auth.json" ]; then
        ln -s "$codexdir/auth.json" "$S/codex/auth.json"
    fi
    CODEX_HOME=$inner/codex
    export CODEX_HOME
    ;;
esac

case $os in
Darwin)
    # SBPL is last-match-wins, which is why the order below is load-bearing:
    #   1. read denies on what is hidden, then what is allowed back from them;
    #   2. a blanket write deny, then the write allows;
    #   3. write denies re-applied LAST on the hidden paths and the target,
    #      so that no allow above can win for them.
    #
    # WRITABLE, each with its reason:
    #   /dev/null /dev/tty /dev/dtracehelper  shells and runtimes open them
    #   S                                     scratch, TMPDIR, the codex state
    #   the target, rw mode only              where an editing agent writes
    #   codex: the real auth.json             so a token refresh persists
    #   claude: the state paths named below   the CLI will not run without them
    #           and its own tool directory under /private/tmp/claude-<uid>
    #           plus the /tmp/claude-<hex>-cwd file its shell tool ends every
    #           command by writing (without it each command exits 1)
    #
    # Deliberately NOT writable: /tmp and /var/tmp (the staged target sits in a
    # host temp directory, and a writable ancestor lets the agent rename it away
    # and put another tree at its path; only S is writable, and it is a sibling).
    # The claude CLI makes its own working directory for its shell tool at
    # /private/tmp/claude-<uid>/<cwd with every non-alphanumeric as ->, and its
    # tool calls fail without it. Only that one directory is opened: the rest of
    # claude-<uid> holds the operator's other live claude sessions' scratch.
    ctmp=/private/tmp/claude-$(id -u)
    cslug=$(printf '%s' "$target" | sed 's/[^A-Za-z0-9]/-/g')
    prof='(version 1)
(allow default)'
    prof="$prof
(deny file-read* (subpath \"$repo\") (subpath \"$fpdir\"))"
    # What each CLI may NOT see of the other's state, and of its own.
    case $cli in
    codex)
        prof="$prof
(deny file-read* (subpath \"$codexdir\") (subpath \"$claudedir\") (literal \"$claudejson\"))
(allow file-read* (literal \"$codexdir/auth.json\"))"
        ;;
    claude)
        prof="$prof
(deny file-read* (subpath \"$codexdir\") (subpath \"$ctmp\"))
(allow file-read* (literal \"$ctmp\"))
(allow file-read* (subpath \"$ctmp/$cslug\"))"
        for d in projects sessions session-env shell-snapshots todos file-history backups history.jsonl; do
            prof="$prof
(deny file-read* (subpath \"$claudedir/$d\"))"
        done
        ;;
    *)
        prof="$prof
(deny file-read* (subpath \"$codexdir\") (subpath \"$claudedir\") (literal \"$claudejson\"))"
        ;;
    esac
    prof="$prof
(deny file-write*)
(allow file-write*
    (literal \"/dev/null\") (literal \"/dev/tty\") (literal \"/dev/dtracehelper\")
    (subpath \"$S\"))"
    case $cli in
    codex)
        prof="$prof
(allow file-write* (literal \"$codexdir/auth.json\"))"
        ;;
    claude)
        # Transcripts and per-session state: written, never read back (denied
        # above). .claude.json and its temp and lock siblings: the CLI rewrites it.
        prof="$prof
(allow file-write* (subpath \"$ctmp/$cslug\") (regex #\"^/private/tmp/claude-[0-9a-f]+-cwd$\") (prefix \"$claudejson\")"
        for d in projects sessions session-env shell-snapshots todos file-history backups telemetry cache; do
            prof="$prof (subpath \"$claudedir/$d\")"
        done
        prof="$prof)"
        ;;
    esac
    if [ "$mode" = rw ]; then
        prof="$prof
(allow file-write* (subpath \"$target\"))"
    fi
    prof="$prof
(deny file-write* (subpath \"$repo\") (subpath \"$fpdir\"))"
    if [ "$mode" = ro ]; then
        prof="$prof
(deny file-write* (subpath \"$target\"))"
    fi
    set -- sandbox-exec -p "$prof" "$@"
    ;;
Linux)
    # POSIX sh has no arrays, so the mount arguments are appended after the
    # agent's argv in "$@" and the agent's argv is then rotated behind them.
    # Order matters, later mounts sit on top of earlier ones. The root is the
    # host's, read-only: an allowlist of /usr /bin /lib* /etc would be tighter
    # but breaks on every host layout that is not the one it was written on, and
    # an ~unnoticed missing library fails every lens identically and looks like
    # the model's fault. So the root stays and what is hidden is the part that
    # holds the operator: /run (the user bus and the system sockets), the home
    # directory, and the checkout.
    argc=$#
    # /run: the user D-Bus socket at /run/user/<uid>/bus lets a process start
    # anything as the operator OUTSIDE this sandbox, and /run/docker.sock is root.
    # /var/run is a symlink to /run. DNS needs the one directory systemd-resolved's
    # /etc/resolv.conf points into.
    set -- "$@" --tmpfs /run
    if [ -d /run/systemd/resolve ]; then
        set -- "$@" --ro-bind /run/systemd/resolve /run/systemd/resolve
    fi
    # The home directory is replaced by an empty tmpfs: ~/.ssh, ~/.aws, ~/.gnupg
    # and every other repository stop existing. Back in, read-only: the places a
    # CLI is installed (nvm, npm, cargo, ~/.local...) and the directory the agent
    # binary itself resolves into.
    set -- "$@" --tmpfs "$home"
    keep=
    for d in .nvm .npm-global .volta .fnm .asdf .bun .cargo .local .deno bin; do
        keep="$keep$nl$home/$d"
    done
    case $agent_path in
    "$home"/*)
        rest=${agent_path#"$home"/}
        keep="$keep$nl$home/${rest%%/*}"
        ;;
    esac
    oldifs=$IFS
    IFS=$nl
    for d in $keep; do
        [ -n "$d" ] && [ -d "$d" ] || continue
        set -- "$@" --ro-bind "$d" "$d"
    done
    IFS=$oldifs
    case $cli in
    codex)
        # Only the credential is bound, writable so a token refresh persists; the
        # fresh CODEX_HOME in S symlinks to it.
        if [ -f "$codexdir/auth.json" ]; then
            set -- "$@" --bind "$codexdir/auth.json" "$codexdir/auth.json"
        fi
        ;;
    claude)
        if [ -d "$claudedir" ]; then
            # The state directories are created in the real directory if they are
            # missing, because a mount point has to exist. Each is then replaced
            # by an empty tmpfs: writable for the session, never persisted, and
            # earlier transcripts are not there to read.
            set -- "$@" --ro-bind "$claudedir" "$claudedir"
            for d in projects sessions session-env shell-snapshots todos file-history backups telemetry cache; do
                mkdir -p "$claudedir/$d"
                set -- "$@" --tmpfs "$claudedir/$d"
            done
            if [ -f "$claudedir/.credentials.json" ]; then
                set -- "$@" --bind "$claudedir/.credentials.json" "$claudedir/.credentials.json"
            fi
        fi
        if [ -f "$claudejson" ]; then
            set -- "$@" --bind "$claudejson" "$claudejson"
        fi
        ;;
    esac
    # /tmp is S: private to this invocation, on the host's disk, removed on exit.
    # It is NOT the host's /tmp, so the staged target's host parent is not
    # writable from here: bound next, the target appears at its own path inside
    # a directory this sandbox made up, and renaming that directory renames
    # nothing on the host.
    set -- "$@" --bind "$S" /tmp
    if [ "$mode" = rw ]; then
        set -- "$@" --bind "$target" "$target"
    else
        set -- "$@" --ro-bind "$target" "$target"
    fi
    # Last, so nothing above can re-expose them.
    set -- "$@" --tmpfs "$repo"
    if [ -d "$fpdir" ]; then
        set -- "$@" --tmpfs "$fpdir"
    fi
    n=0
    while [ "$n" -lt "$argc" ]; do
        a=$1
        shift
        set -- "$@" "$a"
        n=$((n + 1))
    done
    # --dev /dev: a fresh /dev (null, tty, urandom, a private shm). --proc /proc
    # in a NEW pid namespace, or /proc/<fixpoint's pid>/cwd would reach the
    # checkout through the shared /proc. --die-with-parent ties the agent to the
    # wrapper's lifetime.
    set -- bwrap --die-with-parent --unshare-pid --unshare-ipc --ro-bind / / --dev /dev --proc /proc "$@"
    ;;
esac

# The agent is a child, so S can be removed afterwards. Its stdin is the
# wrapper's: a background job otherwise reads /dev/null, and the agents read
# their prompt from stdin.
exec 3<&0
"$@" <&3 3<&- &
child=$!
exec 3<&-
rc=0
while :; do
    wait "$child" && rc=0 || rc=$?
    kill -0 "$child" 2>/dev/null || break
done
exit "$rc"
