#!/bin/sh
# One harness call against a self-served model, BEFORE a vllm@ sweep. It
# answers the two questions a sweep cannot recover from afterwards:
#
#   1. Does the route work end to end -- /v1/messages, tool calls, a reply the
#      harness can read? (A tool call is forced: a reviewer that cannot Read a
#      file reviews nothing.)
#   2. Do the harness's outputTokens include the REASONING tokens? vLLM's
#      Anthropic endpoint has not always returned thinking blocks
#      (vllm-project/vllm#29915). If the usage it reports leaves them out, a
#      thinking-efficient model looks cheaper than it is -- and token
#      efficiency is exactly what ThinkingCap claims. So the harness's count is
#      compared with the server's own counter, vllm:generation_tokens_total.
#
#   VLLM_BASE_URL=http://localhost:8000 VLLM_API_KEY=... \
#       bench/vllm-preflight.sh vllm@bottlecapai/ThinkingCap-Qwen3.8-27B
#
# Exit 0 when both hold. Run it with nothing else using the server: the
# counter is server-wide.
set -eu

MODEL=${1:?usage: bench/vllm-preflight.sh vllm@<hf-repo>}
SERVED=${MODEL#vllm@}
: "${VLLM_BASE_URL:?set VLLM_BASE_URL}" "${VLLM_API_KEY:?set VLLM_API_KEY}"
BASE=${VLLM_BASE_URL%/}

# The server's cumulative generated-token counter. Fetched into a variable,
# not piped: in a pipeline sh reports awk's status, so a failed fetch would
# read as 0 and turn the comparison below into a pass.
generated() {
    m=$(curl -fsS -H "Authorization: Bearer $VLLM_API_KEY" "$BASE/metrics") || {
        echo "preflight: cannot read $BASE/metrics" >&2
        return 1
    }
    printf '%s\n' "$m" | awk '/^vllm:generation_tokens_total/ { s += $NF; n++ }
        END { if (!n) exit 1; printf "%d\n", s }' || {
        echo "preflight: $BASE/metrics has no vllm:generation_tokens_total" >&2
        return 1
    }
}

dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT
word=fixpoint-$$-preflight
echo "The password is $word." >"$dir/note.txt"

before=$(generated) || exit 1
out=$(cd "$dir" && printf 'Read the file note.txt and reply with the password it contains, nothing else.' |
    env -u ANTHROPIC_API_KEY CLAUDE_CODE_EFFORT_LEVEL=xhigh \
        ANTHROPIC_BASE_URL="$BASE" ANTHROPIC_AUTH_TOKEN="$VLLM_API_KEY" \
        ANTHROPIC_DEFAULT_HAIKU_MODEL="$SERVED" ANTHROPIC_DEFAULT_SONNET_MODEL="$SERVED" \
        ANTHROPIC_DEFAULT_OPUS_MODEL="$SERVED" ANTHROPIC_SMALL_FAST_MODEL="$SERVED" \
        CLAUDE_CODE_SUBAGENT_MODEL="$SERVED" \
        claude --model "$SERVED" --output-format json -p --setting-sources user)
after=$(generated) || exit 1

printf '%s' "$out" | python3 -c '
import json, sys
word, server = sys.argv[1], int(sys.argv[2])
r = json.load(sys.stdin)
usage = r.get("modelUsage", {})
harness = sum(m.get("outputTokens", 0) for m in usage.values())
text = (r.get("result") or "").strip()
print(f"reply:   {text[:120]!r}")
print(f"models:  {sorted(usage)}")
print(f"output tokens: harness {harness}, server {server}")
ok = True
if word not in text:
    print("FAIL: the reply does not carry the password -- the Read tool call or the reply parsing is broken")
    ok = False
# 10% slack: the server counter can include a side call the harness keys
# under the same model; a reasoning-blind count is off by far more than that.
if server <= 0 or harness < 0.9 * server:
    print("FAIL: the harness counts fewer output tokens than the server generated -- reasoning is probably missing from usage")
    ok = False
if ok:
    print("OK: tool call works, reply is readable, output tokens include reasoning")
sys.exit(0 if ok else 1)
' "$word" "$((after - before))"
