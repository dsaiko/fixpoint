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
# The goal is BENCH INTEGRITY (the key, past runs, other lenses' findings, writes
# outside the places given), not a defence of the operator against a hostile
# model: the lens still holds its CLI's provider key and the network.
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
#               binaries under temporary dir" and carries on; harmless so far). On a
#               route that authenticates with the operator's ChatGPT login (run.sh
#               says so in bench/.sandbox-codex-auth; no file, no login) the only
#               thing put in it is a SYMLINK to the real ~/.codex/auth.json. A route
#               with its own key (ollama-direct@, codex@ via OpenRouter) is given
#               no login at all. The symlink, not a copy, so no credential is ever
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
#               cannot be modified; projects/, sessions/, file-history/ and the
#               other transcript stores and history.jsonl cannot be read, and on
#               macOS cannot be written either (on Linux they are empty tmpfs, and
#               history.jsonl an empty file). What stays
#               WRITABLE is .claude.json, which is global state that includes
#               user-scope MCP servers: a lens that writes one there plants code
#               that runs the next time the operator starts claude. That cannot
#               be closed while the claude CLI needs to rewrite that file.
#
# WHAT IT DOES NOT ENFORCE (read this before trusting a score)
#   - The agent can still read most of $HOME on macOS (~/.ssh, ~/.aws, other
#     repositories, including other CHECKOUTS of this one, which hold the same
#     manifests), the CLI's own credential, and use the network.
#   - On macOS the operator's terminal is not hidden but its TIOCSTI ioctl is denied
#     (see the profile); the Linux sandbox has a session of its own instead.
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

# The claude config directory's session and transcript state: unreadable (a macOS
# read deny, a Linux tmpfs mask). history.jsonl is a file and is handled beside this.
claude_unreadable="projects sessions session-env shell-snapshots todos file-history backups debug plans paste-cache tasks teams usage-data image-cache"

# Which CLI, from argv[0]. An unknown binary is given no state at all.
cli=${1##*/}
case $cli in
codex | claude) ;;
*) cli=other ;;
esac
agent_path=$(command -v "$1" 2>/dev/null) || agent_path=

# Whether this codex route authenticates with the operator's ChatGPT login. Only
# then is ~/.codex/auth.json given to the lens. A route that carries its own API key
# (ollama-direct@, codex@ via OpenRouter, anything with a model_provider of its own)
# needs no login and gets none. bench/run.sh knows the route and says so in
# bench/.sandbox-codex-auth; the wrapper does not guess from the command line, and no
# file means no credential.
codex_auth=0
if [ "$cli" = codex ] && [ -f "$repo/bench/.sandbox-codex-auth" ]; then
    # The WHOLE file must be exactly "chatgpt" and one newline: a second line or any
    # trailing junk grants nothing (the "x" keeps command substitution from eating the
    # trailing newline that is part of what is compared).
    route=$(cat "$repo/bench/.sandbox-codex-auth"; echo x)
    want=$(printf 'chatgpt\nx')
    [ "$route" != "$want" ] || codex_auth=1
fi

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
# The trap is installed BEFORE the first check that can fail, or a refusal here
# would leave the directory behind.
# A CLI can leave a descendant still writing into S when the agent itself has exited
# (codex's plugin clone did): a few tries, a second apart, so the directory is not
# left behind on that race.
cleanup() {
    n=0
    while [ -e "$S" ] && [ "$n" -lt 5 ]; do
        rm -rf "$S" 2>/dev/null || true
        [ ! -e "$S" ] || sleep 1
        n=$((n + 1))
    done
}
trap cleanup EXIT
resolved=$(phys "$S")
[ -n "$resolved" ] || die "cannot resolve the scratch directory"
S=$resolved
plain "$S"
chmod 700 "$S"

child=
# A signal is forwarded to the agent and the wait below resumes; EXIT removes S
# whichever way the wrapper ends. (A SIGKILL cannot be trapped, which is why no
# credential is ever copied into S: see the codex note above.)
on_signal() {
    [ -z "$child" ] || kill -TERM "$child" 2>/dev/null || true
}
# A signal the wrapper was sent must not be forgotten when the agent, having received
# it, exits 0: the run was interrupted and the exit status says so (128 + the signal).
got_signal=0
trap 'got_signal=143; on_signal' TERM
trap 'got_signal=130; on_signal' INT
trap 'got_signal=129; on_signal' HUP

# Where S is, from inside. On Linux it IS the sandbox's /tmp.
case $os in
Linux) inner=/tmp ;;
*) inner=$S ;;
esac

# Disposable CLI state.
TMPDIR=$inner
export TMPDIR
case $cli in
claude)
    # The claude CLI's shell tool makes /private/tmp/claude-<uid>/<cwd slug>, shared by
    # every lens on this target; with this variable it makes it under S instead.
    CLAUDE_CODE_TMPDIR=$inner
    export CLAUDE_CODE_TMPDIR
    ;;
codex)
    mkdir "$S/codex"
    if [ "$codex_auth" = 1 ] && [ -f "$codexdir/auth.json" ]; then
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
    #   /dev/null /dev/dtracehelper           shells and runtimes open them
    #   S                                     scratch, TMPDIR, the codex state
    #   the target, rw mode only              where an editing agent writes
    #   codex: the real auth.json             so a token refresh persists, and only
    #                                         on a route that authenticates with it
    #   claude: .claude.json and its siblings the CLI rewrites it on every start
    #
    # Other lenses running at the same time have scratch directories of their own
    # beside S: those are unreadable (their codex sessions are their findings), and
    # the claude CLI is pointed at S for its shell-tool directory, so two lenses on
    # the same target do not share /private/tmp/claude-<uid>/<cwd slug>.
    #
    # Deliberately NOT writable: /tmp and /var/tmp (the staged target sits in a
    # host temp directory, and a writable ancestor lets the agent rename it away
    # and put another tree at its path; only S is writable, and it is a sibling).
    prof='(version 1)
(allow default)'
    prof="$prof
(deny file-read* (subpath \"$repo\") (subpath \"$fpdir\") (prefix \"$tb/bench-sandbox.\"))
(allow file-read* (subpath \"$S\"))"
    # What each CLI may NOT see of the other's state, and of its own.
    case $cli in
    codex)
        prof="$prof
(deny file-read* (subpath \"$codexdir\") (subpath \"$claudedir\") (literal \"$claudejson\"))"
        if [ "$codex_auth" = 1 ]; then
            prof="$prof
(allow file-read* (literal \"$codexdir/auth.json\"))"
        fi
        ;;
    claude)
        prof="$prof
(deny file-read* (subpath \"$codexdir\"))"
        for d in $claude_unreadable history.jsonl; do
            prof="$prof
(deny file-read* (subpath \"$claudedir/$d\"))"
        done
        ;;
    *)
        prof="$prof
(deny file-read* (subpath \"$codexdir\") (subpath \"$claudedir\") (literal \"$claudejson\"))"
        ;;
    esac
    # The operator's controlling terminal: without a session of its own (setsid would
    # leak agents past fixpoint's process-group kill on timeout, and macOS has no
    # die-with-parent) the lens inherits it, and TIOCSTI on /dev/tty or on an inherited
    # tty fd types into the operator's shell, outside this profile. Denying the ioctl on
    # the terminal devices stops both, the open /dev/tty and the already-inherited fd.
    prof="$prof
(deny file-ioctl (regex #\"^/dev/(tty|pty|console)\"))"
    prof="$prof
(deny file-write*)
(allow file-write*
    (literal \"/dev/null\") (literal \"/dev/dtracehelper\")
    (subpath \"$S\"))"
    case $cli in
    codex)
        if [ "$codex_auth" = 1 ]; then
            prof="$prof
(allow file-write* (literal \"$codexdir/auth.json\"))"
        fi
        ;;
    claude)
        # .claude.json and its temp and lock siblings: the CLI rewrites it. Nothing
        # else under the config directory is writable: not settings.json, not
        # plugins/, not projects/ (a memory/MEMORY.md planted there is read by the
        # operator's next session), not shell-snapshots/. The CLI runs without them;
        # a lens's transcript is simply not kept.
        prof="$prof
(allow file-write* (prefix \"$claudejson\"))"
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
        if [ "$codex_auth" = 1 ] && [ -f "$codexdir/auth.json" ]; then
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
            for d in $claude_unreadable telemetry cache; do
                mkdir -p "$claudedir/$d"
                set -- "$@" --tmpfs "$claudedir/$d"
            done
            # A file, not a directory, so not a tmpfs: it is replaced by an empty one.
            if [ -f "$claudedir/history.jsonl" ]; then
                set -- "$@" --ro-bind /dev/null "$claudedir/history.jsonl"
            fi
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
    # A TMPDIR other than /tmp holds the other lenses' scratch directories (their
    # findings) beside this one's; it is masked, and the target is bound back below.
    case $tb in
    /tmp | / | "$home") ;;
    *) set -- "$@" --tmpfs "$tb" ;;
    esac
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
    # wrapper's lifetime. --new-session gives it a session of its own: without one it
    # shares the operator's controlling terminal and can inject keystrokes into it
    # (TIOCSTI); the agent is still killed when bwrap goes, by the parent-death signal.
    set -- bwrap --die-with-parent --new-session --unshare-pid --unshare-ipc --ro-bind / / --dev /dev --proc /proc "$@"
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
if [ "$got_signal" != 0 ] && [ "$rc" = 0 ]; then
    rc=$got_signal
fi
exit "$rc"
