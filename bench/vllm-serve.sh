#!/bin/sh
# Serve one open-weight model with vLLM for a vllm@ bench run. Runs ON the
# rented GPU box (a RunPod pod from the PyTorch template, or anything with an
# NVIDIA GPU, Python 3 and internet), not on the machine that runs the bench:
#
#   scp bench/vllm-serve.sh root@<pod>:/workspace/
#   ssh root@<pod> 'VLLM_API_KEY=<secret> sh /workspace/vllm-serve.sh'
#
# Then, from the bench machine, tunnel the port and point run.sh at it:
#
#   ssh -N -L 8000:localhost:8000 root@<pod-ip> -p <pod-ssh-port> &
#   VLLM_BASE_URL=http://localhost:8000 VLLM_API_KEY=<secret> \
#       bench/vllm-preflight.sh vllm@bottlecapai/ThinkingCap-Qwen3.8-27B
#
# The SSH tunnel, not RunPod's https proxy, is the intended path: the proxy
# sits behind a ~100 s gateway timeout that a long prefill can hit before the
# first byte, and a 524 there would be recorded as the candidate failing. So
# the server binds localhost unless HOST says otherwise.
#
# Defaults are the "best version" of ThinkingCap-Qwen3.8-27B: unquantized BF16
# weights (~55 GB, so an 80 GB H100 or a 141 GB H200), the reasoning effort
# its card names as default (xhigh) pinned explicitly, and the parsers the card
# prescribes. Override with MODEL, MAX_LEN, EFFORT, SPEC=1 (MTP speculative
# decoding: faster, distribution-preserving, off by default so the first run
# has one less moving part).
set -eu

MODEL=${MODEL:-bottlecapai/ThinkingCap-Qwen3.8-27B}
MAX_LEN=${MAX_LEN:-262144}
EFFORT=${EFFORT:-xhigh}
PORT=${PORT:-8000}
WORK=${WORK:-/workspace}
: "${VLLM_API_KEY:?set VLLM_API_KEY to a long random secret; the bench sends it as a bearer token}"

# HF_HUB_CACHE, not HF_HOME: only the weights move to $WORK. HF_HOME would
# also move where the hub looks for the login token, and the ThinkingCap
# repos are GATED (accept the license on the model page, then put a read
# token in ~/.cache/huggingface/token or HF_TOKEN) -- a token in the usual
# place would then be ignored and every download 401s.
export HF_HUB_CACHE="$WORK/hf"
# The venv defaults to the container disk, not $WORK: a RunPod volume can be a
# network filesystem (MooseFS on the first pod used), where importing torch and
# vLLM from thousands of small files is slow. The weights stay on $WORK.
VENV=${VENV:-/root/vllm-venv}
LOG="$WORK/vllm.log"

if [ ! -x "$VENV/bin/vllm" ]; then
    echo "vllm-serve: installing vLLM into $VENV"
    # --break-system-packages only as the fallback: Ubuntu 24.04's system
    # Python refuses pip installs without it (PEP 668), older images do not.
    python3 -m pip install -q -U uv 2>/dev/null ||
        python3 -m pip install -q -U --break-system-packages uv
    python3 -m uv venv -q "$VENV"
    # The PyPI wheel follows vLLM's default CUDA, which moved to 13 (driver
    # 580+). The first pod had driver 570 (CUDA 12.8): PyPI's vllm 0.30.0
    # installed, then died on `libcudart.so.13`. On a CUDA 12 driver, install
    # the release's +cu129 wheel instead. --torch-backend, not
    # --extra-index-url: the latter lets the torch index answer for EVERY
    # package, and its stale `packaging` made the resolve unsatisfiable.
    driver_cuda=$(nvidia-smi | sed -n 's/.*CUDA Version: *\([0-9]*\)\..*/\1/p' | head -n 1)
    if [ "${driver_cuda:-0}" -lt 13 ]; then
        v=${VLLM_VERSION:-$(curl -fsS https://api.github.com/repos/vllm-project/vllm/releases/latest |
            python3 -c 'import json, sys; print(json.load(sys.stdin)["tag_name"].lstrip("v"))')}
        python3 -m uv pip install -q --python "$VENV/bin/python" \
            "https://github.com/vllm-project/vllm/releases/download/v$v/vllm-$v+cu129-cp38-abi3-manylinux_2_28_$(uname -m).whl" \
            --torch-backend=cu129
    else
        python3 -m uv pip install -q --python "$VENV/bin/python" --torch-backend=auto vllm
    fi
fi

# FlashInfer JIT-compiles its sampling kernels on first use and shells out to
# `ninja` (installed into the venv) and `nvcc` (the image's CUDA toolkit).
# Neither is on a non-login shell's PATH, and the engine died on the first
# pod with "No such file or directory: 'ninja'".
[ -d /usr/local/cuda/bin ] && export CUDA_HOME=/usr/local/cuda
PATH="$VENV/bin:${CUDA_HOME:+$CUDA_HOME/bin:}$PATH"
export PATH

# Pin the weights to the commit that exists NOW and print it: a repo can be
# re-pushed under the same name, and a measurement of unknown weights is not a
# measurement (the bench records `ollama show` for the same reason).
REVISION=${REVISION:-$("$VENV/bin/python" -c "
from huggingface_hub import HfApi
print(HfApi().model_info('$MODEL').sha)")}
echo "vllm-serve: $MODEL @ $REVISION"
echo "vllm-serve: vllm $("$VENV/bin/python" -c 'import vllm; print(vllm.__version__)')"
nvidia-smi --query-gpu=name,memory.total,driver_version --format=csv,noheader

SPEC_ARGS=
if [ "${SPEC:-0}" = 1 ]; then
    SPEC_ARGS='--speculative-config {"method":"mtp","num_speculative_tokens":3}'
fi

# A server already on the port would answer the readiness wait below for
# the one being started -- which then dies on the busy port or the GPU the old
# one holds, and the sweep measures the OLD settings under the new notes.
if curl -fsS "http://localhost:$PORT/health" >/dev/null 2>&1; then
    echo "vllm-serve: something already serves on port $PORT; stop it first (pkill -f 'vllm serve')" >&2
    exit 1
fi

# --served-model-name is the bare repo id: run.sh strips vllm@ and checks the
# server lists exactly that. --language-model-only: the reviewers send text
# only, and skipping the vision inputs leaves that memory to the KV cache.
# --enable-prompt-tokens-details reports cached prompt tokens, which the
# harness reads as cacheReadInputTokens. Bound to localhost by default: the
# SSH tunnel is the way in, and --api-key does not cover /metrics (vLLM's own
# docs: it guards /v1, /v2 and /inference only). Everything else is the model
# card's own command.
# shellcheck disable=SC2086 # SPEC_ARGS is two words on purpose
nohup "$VENV/bin/vllm" serve "$MODEL" \
    --revision "$REVISION" \
    --served-model-name "$MODEL" \
    --host "${HOST:-127.0.0.1}" --port "$PORT" \
    --api-key "$VLLM_API_KEY" \
    --max-model-len "$MAX_LEN" \
    --gpu-memory-utilization 0.92 \
    --reasoning-parser qwen3 \
    --enable-auto-tool-choice --tool-call-parser qwen3_xml \
    --default-chat-template-kwargs "{\"reasoning_effort\":\"$EFFORT\"}" \
    --language-model-only \
    --enable-prompt-tokens-details \
    $SPEC_ARGS \
    >"$LOG" 2>&1 &
PID=$!
echo "vllm-serve: started (pid $PID), log $LOG; first start downloads the weights"

# Wait for /health, failing fast if the server dies (the usual cause: MAX_LEN
# does not fit the KV cache, and the log names the length that would).
until curl -fsS "http://localhost:$PORT/health" >/dev/null 2>&1; do
    if ! kill -0 "$PID" 2>/dev/null; then
        tail -n 30 "$LOG" >&2
        echo "vllm-serve: server exited; see $LOG (lower MAX_LEN if the KV cache did not fit)" >&2
        exit 1
    fi
    sleep 10
done
if ! kill -0 "$PID" 2>/dev/null; then
    echo "vllm-serve: /health answered but pid $PID is gone; see $LOG" >&2
    exit 1
fi
echo "vllm-serve: ready"
curl -fsS -H "Authorization: Bearer $VLLM_API_KEY" "http://localhost:$PORT/v1/models"
echo
