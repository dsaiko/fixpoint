# Agents

One file per agent, referenced from a task config by bare name:

    roles:
      coder: { agent: claude-coder, prompt: fix }
      review:
        agents: [codex, claude, kimi, glm]

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

## effort (claude)

Reasoning budget per request; all five levels verified with `claude -p`:

    low     minimal thinking; fastest and cheapest, mechanical tasks only
    medium  light reasoning; quick checks, simple targets
    high    substantial reasoning; solid default for review passes
    xhigh   deep reasoning; complex fixes, subtle bugs
    max     unbounded; hardest problems, slowest and priciest

Higher effort buys judgment at the cost of latency. Reviewers run every round, so
their effort multiplies across the loop; the coder runs once per round.

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
protection for that agent. `--check-live` invokes every agent, so a missing
variable surfaces there rather than mid-run.

## A note on prompt_via

`stdin` keeps the prompt off the process argument list. `arg` puts the entire
prompt — including the reviewed material, which in git-diff or pr mode may BE the
credential under review — on argv, where any local user can read it via `ps` or
/proc. fixpoint warns at run start for every agent using `arg`. Prefer `stdin`
whenever the CLI supports it.

The `command` values here are starting points: verify the flags your installed
version of each CLI expects.
