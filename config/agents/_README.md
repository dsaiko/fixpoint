# Agents

One file per agent, referenced by bare name. The reviewer POOL lives in
`defaults.yaml` so that changing who reviews is one edit rather than the same edit
in every config; a task config names its coder and its lenses, and inherits the
pool:

    # defaults.yaml
    roles:
      review:
        agents: [claude, kimi-ollama, deepseek-ollama]

    # fix-code.yaml
    extends: defaults
    roles:
      coder: { agent: claude-coder, prompt: fix }
      review:
        strategy: rotate
        prompts: [review-bugs, review-security]

Setting `agents` in a task config REPLACES the inherited list rather than adding
to it — `review-pr` does that deliberately, narrowing the panel for untrusted code.

An agent is a command that receives a prompt and prints text to stdout. That is
the whole provider abstraction — the orchestrator wraps the prompt with role
instructions plus a required JSON schema and extracts the JSON from whatever the
command prints, so any CLI works with no code change.

`model` and `effort` are injected through the `{{model}}` and `{{effort}}`
placeholders. A command token whose placeholder resolves to empty is dropped
whole, so keep a flag and its value in ONE token ("--model {{model}}") and
`effort` disappears cleanly for providers that have no such setting.

`can_edit` must reflect what the command ACTUALLY permits. Reviewers run in
enforced read-only modes (`can_edit: false`); only an agent whose command allows
writing files may be assigned as `roles.coder`, and validation enforces that.

A read-only claim the command contradicts is rejected: `can_edit: false` fails
validation when the command passes a flag that grants the write tools. That
covers the flags which auto-approve every tool request
(`--dangerously-skip-permissions`, `--dangerously-bypass-approvals-and-sandbox`,
`--yolo`, `--full-auto`, `--permission-mode bypassPermissions`) AND the modes
that auto-approve only the edits (`--permission-mode acceptEdits`,
`--sandbox workspace-write`, `--sandbox danger-full-access`,
`--mode accept-edits`, `--approval-mode auto_edit`) — an edits-only mode is no
safe middle ground here, since writing files is precisely what `can_edit: false`
denies. Earn read-only from the command itself — omit the flag, or use an
enforced sandbox like `codex --sandbox read-only` — or declare `can_edit: true`,
which keeps the agent out of reviewer pools. A plan-mode flag alone
(`--mode plan`) is not enough either: fixpoint cannot verify it, and a reviewer
reading untrusted code is exactly where an unenforced claim fails.

## effort (claude)

Reasoning budget per request; all five levels verified with `claude -p`:

    low     minimal thinking; fastest and cheapest, mechanical tasks only
    medium  light reasoning; quick checks, simple targets
    high    substantial reasoning; solid default for review passes
    xhigh   deep reasoning; complex fixes, subtle bugs
    max     unbounded; hardest problems, slowest and priciest

Higher effort buys judgment at the cost of latency. Reviewers run every round, so
their effort multiplies across the loop; the coder runs once per round.

The heading says **(claude)** because the setting is Anthropic's, not the
harness's. `--effort` becomes a `thinking.budget_tokens` field in the API request,
so it only does something if whatever serves the model reads that field.

**ollama does not.** Setting `effort` on an ollama-served agent is accepted,
validates, runs without error — and changes nothing, which is worse than being
rejected, because the config then claims a reviewer runs deliberately that the
panel never configured. Measured against `/v1/messages` on the local ollama proxy
(the endpoint `ollama launch claude` points the harness at):

    model            thinking budget    thinking produced
    glm-5.2          not sent           ~1511 tok
    glm-5.2          1024               ~1487 tok    45% over budget
    kimi-k2.7-code   not sent           ~3443 tok
    kimi-k2.7-code   1024               ~3428 tok    3.3x over budget

Two findings there: both models emit a thinking block whether or not one is asked
for, and a budget that should bind is ignored outright — the with/without pairs
differ by under half a percent. So `kimi-ollama.yaml` and `glm-ollama.yaml`
deliberately declare no `effort`, and adding one is not a knob, it is a comment
that lies. To change how hard those models work, change `model` or the prompt.

## Naming: the route is part of the identity

An agent whose name carries a provider suffix — `kimi-ollama`, `kimi-openrouter` —
is the same MODEL reached a different way, and the two are not interchangeable.
The suffix exists because the route changes behaviour that no other field records:

    ollama       drops cache_control and thinking budgets; 0% cache rate
    openrouter   honours both; ~99% cache rate through the same harness

Measured on one fix-branch run, the two ollama-routed reviewers spent 41.6M fresh
input tokens across nine sessions while claude spent 0.4M across five, because
every turn of an uncached agentic session re-pays for the whole conversation. The
same request sent to both proxies returns `in=5418` with no cache fields from
ollama, and `in=16, cacheRead=5604` on the second call from OpenRouter.

Both files are kept for every model that has both routes, and switching is one
word in the panel.

The panel currently runs the `-ollama` one, which is the counter-intuitive choice
and worth the sentence: caching cut the price per token 4.9x and did not cut the
BILL, because the volume is what is wrong. kimi takes ~168 turns to claude's ~37
on the same round, so one cached session still ran ~$5 through OpenRouter. Prepaid
quota absorbs that volume; a card bills it. The quota ceiling is the cost of that
choice — it ended two runs early — which is why glm came off the panel rather than
being moved to a cheaper route.

## env

Each agent receives a non-secret baseline (`PATH`, `HOME`, temp dir, locale, TLS
trust, proxy settings) plus only what it declares:

    env:
      pass: [ANTHROPIC_API_KEY]   # inherit by name, if set in fixpoint's env
      set:  {NO_COLOR: "1"}       # literal values; override anything inherited

Everything else is absent from the process, so an exported secret the agent never
asked for cannot be quoted into a finding. A declared name that is not set is
simply absent — `claude` and `codex` read credentials from `~/.claude` and
`~/.codex` after an interactive login, so in that case the agent needs no
credential in its environment at all.

If a CLI misbehaves after you add it, check whether it needs a variable you have
not declared; `env.inherit_all: true` is the escape hatch, and gives up the
protection for that agent — usable only from a bundle outside the reviewed target,
since an agent file the target ships must not be able to hand itself every
exported secret. `--check-live` invokes every agent, so a missing
variable surfaces there rather than mid-run.

## A note on prompt_via

`stdin` keeps the prompt off the process argument list. `arg` puts the entire
prompt — including the reviewed material, which in git-diff or pr mode may BE the
credential under review — on argv, where any local user can read it via `ps` or
/proc. fixpoint warns at run start for every agent using `arg`. Prefer `stdin`
whenever the CLI supports it.

The `command` values here are starting points: verify the flags your installed
version of each CLI expects.
