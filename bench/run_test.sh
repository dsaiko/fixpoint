#!/bin/sh
# Tests for bench/run.sh's vllm@ route: the guards that keep a credential on
# our own server and refuse a sweep the server cannot serve. Also the
# ollama-codex@ / ollama-direct@ routes' template choice and #<effort> suffix.
# Run by `make bench-check`.
#
# run.sh runs in a scratch copy of the tree it needs, with `curl` and `make`
# stubbed on PATH: curl answers /v1/models from STUB_MODELS (or fails), and
# make -- the first thing run.sh does after writing the candidate agent --
# records the environment and the candidate, then stops the run. Nothing
# reaches a network, the real lock, or the real config/agents/.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/tree/bench" "$tmp/tree/config/agents" "$tmp/bin"
cp "$here/run.sh" "$here/agent-template-vllm.yaml" "$here/agent-template-ollama-codex.yaml" \
    "$here/agent-template-ollama-direct.yaml" "$here/agent-template-openrouter-codex.yaml" "$tmp/tree/bench/"

cat >"$tmp/bin/curl" <<'EOF'
#!/bin/sh
[ "${STUB_CURL_FAIL:-0}" = 1 ] && exit 7
printf '%s' "$STUB_MODELS"
EOF
cat >"$tmp/bin/make" <<'EOF'
#!/bin/sh
{
    echo "AUTH=${ANTHROPIC_AUTH_TOKEN-<unset>}"
    echo "APIKEY=${ANTHROPIC_API_KEY-<unset>}"
} >"$STUB_OUT/env"
cp config/agents/bench-candidate.yaml "$STUB_OUT/candidate.yaml"
exit 3
EOF
chmod +x "$tmp/bin/curl" "$tmp/bin/make"

MODEL=vllm@bottlecapai/ThinkingCap-Qwen3.8-27B
REPO=bottlecapai/ThinkingCap-Qwen3.8-27B
fails=0

# run <expected exit> <label> [VAR=value ...]: run.sh with the stubs, in a
# clean env carrying only what the case sets.
run() {
    want=$1 label=$2
    shift 2
    rm -rf "$tmp/out"
    mkdir "$tmp/out"
    set +e
    env -i PATH="$tmp/bin:/usr/bin:/bin" HOME="$tmp" STUB_OUT="$tmp/out" "$@" \
        sh "$tmp/tree/bench/run.sh" "$MODEL" go >"$tmp/out/log" 2>&1
    got=$?
    set -e
    if [ "$got" != "$want" ]; then
        echo "FAIL $label: exit $got, want $want"
        sed 's/^/    /' "$tmp/out/log"
        fails=$((fails + 1))
        return 1
    fi
    if [ -e "$tmp/tree/bench/.run.lock" ] || [ -e "$tmp/tree/config/agents/bench-candidate.yaml" ]; then
        echo "FAIL $label: left the lock or the candidate behind"
        fails=$((fails + 1))
        return 1
    fi
}

# expect <label> <file> <grep -E pattern> [!]: the file must (or, with !,
# must not) match.
expect() {
    if grep -Eq -- "$3" "$2"; then hit=1; else hit=0; fi
    if [ "${4:-}" = "!" ]; then hit=$((1 - hit)); fi
    if [ "$hit" != 1 ]; then
        echo "FAIL $1"
        fails=$((fails + 1))
    fi
}

GOOD="{\"object\":\"list\",\"data\":[{\"id\":\"$REPO\",\"root\":\"$REPO\"}]}"

# (a) No server named: refused before anything is generated.
run 2 "missing VLLM_BASE_URL" VLLM_API_KEY=k || true
run 2 "missing VLLM_API_KEY" VLLM_BASE_URL=http://vllm.test:8000 || true
[ -e "$tmp/out/env" ] && { echo "FAIL missing vars: make ran"; fails=$((fails + 1)); }

# (b) Server unreachable.
run 2 "curl fails" VLLM_BASE_URL=http://vllm.test:8000 VLLM_API_KEY=k STUB_CURL_FAIL=1 || true

# (c) The weights are loaded but served under ANOTHER name: the repo id
# appears only as `root`, and every session would 404.
run 2 "repo only as root" VLLM_BASE_URL=http://vllm.test:8000 VLLM_API_KEY=k \
    STUB_MODELS="{\"data\":[{\"id\":\"other-name\",\"root\":\"$REPO\"}]}" || true

# (d) The served id matches: the candidate points at OUR server, never at
# OpenRouter, and names the bare repo id.
if run 3 "served model" VLLM_BASE_URL=http://vllm.test:8000/ VLLM_API_KEY=k STUB_MODELS="$GOOD"; then
    expect "candidate points at VLLM_BASE_URL" "$tmp/out/candidate.yaml" '^    ANTHROPIC_BASE_URL: http://vllm\.test:8000$'
    expect "candidate never names openrouter" "$tmp/out/candidate.yaml" 'openrouter' !
    expect "candidate model is the bare repo id" "$tmp/out/candidate.yaml" "^model: $REPO\$"
    expect "no placeholder survives" "$tmp/out/candidate.yaml" '__(MODEL|BASE_URL)__' !
fi

# (e) Credentials already exported are REPLACED, never defaulted: the
# Anthropic token and key must not travel to the GPU box.
if run 3 "exported credentials" VLLM_BASE_URL=http://vllm.test:8000 VLLM_API_KEY=k STUB_MODELS="$GOOD" \
    ANTHROPIC_AUTH_TOKEN=leak ANTHROPIC_API_KEY=leak2; then
    expect "auth token overwritten with VLLM_API_KEY" "$tmp/out/env" '^AUTH=k$'
    expect "ANTHROPIC_API_KEY unset" "$tmp/out/env" '^APIKEY=<unset>$'
fi

# (f) ollama-direct@ without a key: refused before anything is generated.
MODEL=ollama-direct@glm-5.3-flash:cloud
run 2 "direct without OLLAMA_API_KEY" || true
[ -e "$tmp/out/env" ] && { echo "FAIL direct without key: make ran"; fails=$((fails + 1)); }

# (g) ollama-direct@ with a key and an effort suffix: the direct template, the
# bare tag as the model, the suffix as the effort.
MODEL='ollama-direct@glm-5.3-flash:cloud#max'
if run 3 "direct with key" OLLAMA_API_KEY=k; then
    expect "direct candidate points at ollama.com" "$tmp/out/candidate.yaml" 'base_url=https://ollama\.com/v1$'
    expect "direct candidate model is the bare tag" "$tmp/out/candidate.yaml" '^model: glm-5\.3-flash:cloud$'
    expect "direct candidate effort is the suffix" "$tmp/out/candidate.yaml" '^effort: max '
    expect "direct candidate keeps its key out of the model's shell" "$tmp/out/candidate.yaml" 'shell_environment_policy\.exclude=\["OLLAMA_API_KEY"\]'
fi

# (g2) The OpenRouter codex route reads a card-billed key through env_key; codex
# would hand it to the model's shell unless the exclude names it.
MODEL='codex@google/gemini-3.7-flash'
if run 3 "openrouter codex" OPENROUTER_API_KEY=k; then
    expect "openrouter candidate keeps its key out of the model's shell" "$tmp/out/candidate.yaml" 'shell_environment_policy\.exclude=\["OPENROUTER_API_KEY"\]'
fi

# (h) ollama-codex@ takes the same suffix, on its own template, and needs no key.
MODEL='ollama-codex@glm-5.3-flash:cloud#max'
if run 3 "launch with suffix"; then
    expect "launch candidate runs ollama launch" "$tmp/out/candidate.yaml" '^  - launch$'
    expect "launch candidate never names ollama.com" "$tmp/out/candidate.yaml" 'ollama\.com' !
    expect "launch candidate effort is the suffix" "$tmp/out/candidate.yaml" '^effort: max '
fi
MODEL=ollama-codex@glm-5.3-flash:cloud
if run 3 "launch without suffix"; then
    expect "launch candidate keeps xhigh" "$tmp/out/candidate.yaml" '^effort: xhigh '
fi

# (i) An effort no route accepts is refused.
MODEL='ollama-codex@glm-5.3-flash:cloud#ultra'
run 2 "unknown effort" || true

if [ "$fails" -ne 0 ]; then
    echo "bench/run_test.sh: $fails failure(s)" >&2
    exit 1
fi
echo "bench/run_test.sh: vllm@, ollama-codex@ and ollama-direct@ route guards OK"
