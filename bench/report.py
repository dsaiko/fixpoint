#!/usr/bin/env python3
"""Combine bench/results.csv into one score out of 100 per model.

Usage: bench/report.py [results.csv]

The benchmark's 100 points are split across two tasks -- 60 seeded defects in
the code target, 40 seeded flaws in the design target -- so a model's score is
only complete once both have run. Repeats are shown separately and then as a
median, because single-run recall on a hundred seeds is noisy and the variance
is itself information: a model that scores 61 then 38 is worse than one that
scores 49 twice.

A row missing one of the two tasks is reported as incomplete rather than
silently scored out of 60.
"""

import csv
import pathlib
import statistics
import sys

TASK_POINTS = {"code": 60, "design": 40}
AGENTS = pathlib.Path(__file__).resolve().parent.parent / "config" / "agents"

# (input, output, cache-read) $/MTok. Two kinds of row live here and the
# difference matters when reading the table:
#
#   - OpenRouter models bill a card, so their figure is an invoice.
#   - claude and codex run on a subscription, so nothing is billed per token.
#     Their figure is NOTIONAL -- what the same run would have cost at API
#     rates -- and is printed with a leading ~. It is the only way to compare a
#     seat paid in subscription against one paid in credits.
#
# Ollama cloud models are deliberately absent: their plan has no per-token price
# at all, only a usage level drawing on a weekly quota, so a dollar figure for
# them would be fiction. They are priced in tokens instead.
#
# Anthropic rates: platform.claude.com/docs/en/about-claude/pricing, 2026-08-20
# (cache read is the documented 0.1x of base input). gpt-5.6-sol: $5/$30, cached
# input at 10%. OpenRouter rates from each model's own page, same date.
RATES = {
    "claude-opus-5": (5.00, 25.00, 0.50),
    "claude-opus-4-8": (5.00, 25.00, 0.50),
    "claude-fable-5": (10.00, 50.00, 1.00),
    "gpt-5.6-sol": (5.00, 30.00, 0.50),
    "x-ai/grok-4.6": (2.00, 6.00, None),
    "qwen/qwen3.8-27b": (0.40, 3.00, 0.04),
    "qwen/qwen3.8-max": (2.00, 6.00, None),
}

# Who pays, checked against ollama.com on 2026-08-20. Ollama cloud models are
# covered by the Pro/Max subscription and differ only in how fast they draw the
# weekly quota, with ONE exception: kimi-k3, whose library page says it
# "requires a Pro or Max subscription, and consumes extra usage credits" -- a
# real invoice on top of the plan, which no other swept model has. Re-check when
# a model is added; this is an external fact with no API behind it.
OLLAMA_CREDIT_MODELS = ("kimi-k3",)


def agent_model(model):
    """The model id behind a candidate name, for pricing and for display.

    `claude` and `codex` are agent names: what they run is whatever their yaml
    says today, so a row that only says "claude" stops being a measurement the
    moment that file changes.
    """
    path = AGENTS / f"{model}.yaml"
    if not path.exists():
        return None, None
    name, effort = None, None
    for line in path.read_text().splitlines():
        if line.startswith("model:"):
            name = line.split(":", 1)[1].split("#")[0].strip()
        elif line.startswith("effort:"):
            effort = line.split(":", 1)[1].split("#")[0].strip()
    return name, effort


def resolve(model):
    """Display name for the model actually measured."""
    name, effort = agent_model(model)
    if name is None:
        return model
    return f"{name} ({effort})" if effort else name


def route(model):
    """Which harness and provider the bench used, mirroring bench/run.sh.

    It matters for reading the token column: the ollama route does not cache, so
    its input is paid fresh every session, while the claude and OpenRouter
    routes bill a cached prompt at a fraction. Two rows with the same token
    count are not the same spend.
    """
    if (AGENTS / f"{model}.yaml").exists():
        return "cli"
    if model.startswith("codex@"):
        return "openrouter/codex"
    if model.startswith("claude-"):
        return "anthropic"
    if "/" in model:
        return "openrouter"
    return "ollama"


def billing(model, harness):
    if harness.startswith("openrouter"):
        return "card"
    if harness == "ollama":
        base = model.split(":", 1)[0]
        return "sub+credits" if base in OLLAMA_CREDIT_MODELS else "sub (ollama)"
    if model == "codex":
        return "sub (chatgpt)"
    return "sub (claude)"


def cost_per_point(model, harness, tok_in, tok_out, cache_read, points):
    """What one seeded defect cost, in the currency that route actually spends.

    A dollar figure wherever a published rate exists, prefixed with ~ when the
    route is a subscription and the money is therefore notional. Ollama has no
    per-token price, so its seats are priced in thousands of tokens per point --
    the quota drawdown, which is the thing that is actually scarce there.
    """
    if not points:
        return "-"
    name, _ = agent_model(model)
    rates = RATES.get(name or model)
    if rates is None:
        return f"{(tok_in + tok_out) / 1000 / points:.1f}k"
    rate_in, rate_out, rate_cache = rates
    usd = (tok_in * rate_in
           + cache_read * (rate_in if rate_cache is None else rate_cache)
           + tok_out * rate_out) / 1e6
    prefix = "" if harness.startswith("openrouter") else "~"
    return f"{prefix}${usd / points:.3f}"


def main():
    path = pathlib.Path(sys.argv[1] if len(sys.argv) > 1
                        else pathlib.Path(__file__).parent / "results.csv")
    if not path.exists():
        sys.exit(f"{path}: no results yet")
    rows = list(csv.DictReader(path.open()))
    if not rows:
        sys.exit(f"{path}: no measurements yet")

    runs = {}
    for r in rows:
        key = (r["model"], r["repeat"])
        runs.setdefault(key, {})[r["task"]] = r

    per_model = {}
    for (model, repeat), tasks in sorted(runs.items()):
        points = sum(int(t["matched_seeds"]) for t in tasks.values())
        possible = sum(TASK_POINTS.get(task, int(t["seeds"])) for task, t in tasks.items())
        per_model.setdefault(model, []).append({
            "repeat": repeat,
            "points": points,
            "possible": possible,
            "tokens": sum(int(t["tokens_in"]) + int(t["tokens_out"]) for t in tasks.values()),
            "tok_in": sum(int(t["tokens_in"]) for t in tasks.values()),
            "tok_out": sum(int(t["tokens_out"]) for t in tasks.values()),
            "cache_read": sum(int(t["cache_read"]) for t in tasks.values()),
            "errors": sum(int(t["errors"]) for t in tasks.values()),
            "seconds": sum(int(t["duration_s"]) for t in tasks.values()),
            "missing": sorted(set(TASK_POINTS) - set(tasks)),
        })

    ranked = sorted(per_model.items(),
                    key=lambda kv: -statistics.median(r["points"] for r in kv[1]))

    print(f"{'candidate':30s} {'model measured':22s} {'route':11s} {'pays':13s} "
          f"{'score':>9s} {'tok':>9s} {'pts/Mtok':>9s} {'per point':>10s} "
          f"{'sec':>6s} {'err':>4s}")
    print("-" * 141)
    for model, entries in ranked:
        median = statistics.median(e["points"] for e in entries)
        for e in entries:
            mtok = e["tokens"] / 1e6
            eff = e["points"] / mtok if mtok else 0.0
            flag = ""
            if e["missing"]:
                flag = f"  INCOMPLETE (no {', '.join(e['missing'])} run)"
            label = model if len(entries) == 1 else f"{model} #{e['repeat']}"
            measured = resolve(model)
            harness = route(model)
            print(f"{label:30s} {(measured if measured != model else '-'):22s} "
                  f"{harness:11s} {billing(model, harness):13s} "
                  f"{e['points']:>4d}/{e['possible']:<4d} {e['tokens']:>9d} {eff:>9.1f} "
                  f"{cost_per_point(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], e['points']):>10s} "
                  f"{e['seconds']:>6d} {e['errors']:>4d}{flag}")
        if len(entries) > 1:
            print(f"{'  median':30s} {'':22s} {'':11s} {'':13s} {median:>4.0f}/100")
    print()
    print(f"{len(ranked)} model(s); 100 points = {TASK_POINTS['code']} code seeds "
          f"+ {TASK_POINTS['design']} design seeds")
    print("per point: cost of one seeded defect found. $ = billed to a card, "
          "~$ = notional (subscription,")
    print("           nothing billed per token), k = thousands of tokens "
          "(ollama sells quota, not tokens)")


if __name__ == "__main__":
    main()
