#!/usr/bin/env python3
"""Combine bench/results.csv into one score per model, out of the live seed pool.

Usage: bench/report.py [results.csv]
       bench/report.py [results.csv] --html <out.html>
       bench/report.py [results.csv] --markdown <file.md>   (rewrites in place)
       bench/report.py [results.csv] --json <out.json>       (the standings as data)

The benchmark's points are split across two calibrated TARGETS -- the Go target
and the design target -- so a model's score is
only complete once both have run. Further language targets are reported in the
by-target matrix rather than folded into that total: a score is comparable only
to runs sharing its scale. Repeats are shown separately and then as a
median, because single-run recall on a hundred seeds is noisy and the variance
is itself information: a model that scores 61 then 38 is worse than one that
scores 49 twice.

A row missing any scored target is reported as incomplete rather than
silently scored out of one target's pool. The denominator comes from each row's
own `seeds` column, so a retired seed changes the scale everywhere at once.
"""

import csv
import json
import pathlib
import re
import statistics
import sys

# Each row carries its own seed total, and that is what the score is out of --
# NOT a constant here. Retiring a seed changes the live pool, and a hardcoded
# denominator would keep reporting a score out of a scale that no longer
# exists. Kept only as the documented shape of the calibrated pair at the time
# it was calibrated.
CALIBRATED_2026_08_20 = {"go": 60, "design": 40}

# The calibrated pair the published /100 used to mean. Kept because the seat
# decisions on record quote it, and because the by-target matrix still orders
# these two first.
LEGACY_SCALE = ("go", "design")

# What the headline score is out of, as of 2026-09-02: every target, not the
# calibrated pair.
#
# The pair was the whole benchmark when it was calibrated. With seven languages
# measured it became the least representative slice available, and measurably
# so -- against the all-target order it misranked six of twenty models by three
# places or more. nemotron-3-ultra was thirteenth on go+design and seventh on
# all eight, because go and design happen to be its two weakest targets; the
# seated minimax-m3 read tenth and is fifteenth. Two targets out of eight
# cannot carry a headline once the other six disagree with them.
#
# Order is the reading order of the matrix, not a ranking.
SCORED_TARGETS = ("go", "design", "rust", "java", "typescript", "csharp",
                  "cpp", "python")
AGENTS = pathlib.Path(__file__).resolve().parent.parent / "config" / "agents"

# (input, output, cache-read) $/MTok. Two kinds of row live here and the
# difference matters when reading the table:
#
#   - OpenRouter models are paid from prepaid credits, so their figure is a
#     real invoice.
#   - claude and codex run on a subscription, so nothing is billed per token.
#     Their figure is NOTIONAL -- what the same run would have cost at API
#     rates -- and is printed with a leading ~. It is the only way to compare a
#     seat paid in subscription against one paid in credits.
#
# Ollama models are NOT here -- not because they lack a price (they have had one
# since 2026-09-01) but because they live in OLLAMA_RATES below, keyed by the
# name ollama's own price list uses rather than by the tag the bench invokes.
#
# Anthropic rates: platform.claude.com/docs/en/about-claude/pricing, 2026-08-20
# (cache read is the documented 0.1x of base input). gpt-5.6-sol: $5/$30, cached
# input at 10%. OpenRouter rates from each model's own page, same date.
RATES = {
    "claude-opus-5": (5.00, 25.00, 0.50),
    # Opus 5.5, added 2026-09-22 when it was benched. CHEAPER than the Opus 5 it
    # succeeds on every axis -- $4/$20 against $5/$25, and cached input $0.20
    # against $0.50, a 60% cut on the line that dominates a sweep (the claude
    # route caches ~98% of its input). These three numbers are not quoted from a
    # price page: they are SOLVED from a billed call and reproduce its cost
    # exactly. Probe 2026-09-22, claude-opus-5-5, from the CLI's own modelUsage:
    #   2 in, 4 out, 21641 cache read, 14199 cache write (1h), costUSD 0.1180082
    #   2*4 + 4*20 + 21641*0.20 + 14199*8.00, /1e6  =  0.1180082  exactly
    # The 1h cache-write rate ($8.00 = 2x input) is not in this tuple because
    # run_cost does not bill cache writes at all; it is recorded here because it
    # is what pinned the other three -- no other (in, out, cached) triple
    # reproduces that cent-exact total.
    "claude-opus-5-5": (4.00, 20.00, 0.20),
    "claude-opus-4-8": (5.00, 25.00, 0.50),
    "claude-fable-5": (10.00, 50.00, 1.00),
    # Fable 5.1: same input/output as Fable 5, but cached input drops 75%
    # ($1.00 -> $0.25), announced 2026-09-01. On a bench sweep the claude
    # route caches ~98%, so that cut is most of what a run actually pays.
    "claude-fable-5-1": (10.00, 50.00, 0.25),
    # Sonnet 5, added 2026-09-05 when it was benched. The cheapest Anthropic
    # entry here by a distance -- $2/$10 against Opus's $5/$25 -- which is the
    # whole reason to measure it: the claude route caches ~98% of a sweep, so
    # the $0.20 cached-input rate is most of what it would actually pay.
    "claude-sonnet-5": (2.00, 10.00, 0.20),
    # NOTE 2026-09-07: developers.openai.com now shows sol at $4/$20, cached
    # $0.40, footnoted as "promotional pricing ... at least through November 21,
    # 2026". The row keeps the list price it was measured at so the codex
    # figures do not move under a promotion; revisit if the promo becomes list.
    "gpt-5.6-sol": (5.00, 30.00, 0.50),
    # GPT-6 Sol, added 2026-09-22 when it was benched: the successor of the model
    # holding the ChatGPT seat, and cheaper than it on every line -- $2/$10 with
    # cached input at $0.20, against gpt-5.6-sol's $5/$30/$0.50 below.
    #
    # THE COMPARISON BETWEEN THOSE TWO ROWS IS NOT LIKE FOR LIKE, and the note on
    # gpt-5.6-sol says why: that row deliberately keeps LIST price while OpenAI
    # discounts the model to $4/$20 "at least through November 21, 2026". Two
    # independent sources on 2026-09-22 give these figures as PERMANENT, not
    # promotional, so the honest reading of a gpt-6-sol / codex cost gap is
    # against list on one side and a permanent price on the other. Against the
    # promo the input gap is 2x rather than 2.5x. Long-context rows (>272K input)
    # are 2x in / 1.5x out, which no bench session reaches.
    "gpt-6-sol": (2.00, 10.00, 0.20),
    # GPT-5.6 Terra, from developers.openai.com on 2026-09-07 when it was added
    # as a candidate: $2/$12, cached $0.20 -- the cheap sibling of the codex
    # seat, and 20% of astra's list price. Long-context (>272K input) rows are
    # 2x in / 1.5x out, which no bench session reaches.
    "gpt-5.6-terra": (2.00, 12.00, 0.20),
    # GPT-6 Astra, from developers.openai.com on 2026-09-05, the day the model
    # became reachable on ChatGPT-account auth. Fable-5-class list price, and
    # 2x the input of the gpt-5.6-sol it is benched against. Requests over 272K
    # INPUT tokens are surcharged (2x in, 1.5x out); a bench session's own input
    # is nowhere near that, so the plain rate is the right one here.
    "gpt-6-astra": (10.00, 50.00, 1.00),
    "x-ai/grok-4.6": (2.00, 6.00, None),
    "qwen/qwen3.8-27b": (0.40, 3.00, 0.04),
    "qwen/qwen3.8-max": (2.00, 6.00, None),
}

# Ollama per-token rates, $/MTok as (input, output, cached input), from
# ollama.com/pricing on 2026-09-01.
#
# THIS TABLE DID NOT EXIST BEFORE 2026-09-01, when ollama published per-token
# rates for Pro/Max/Team.
#
# CORRECTED 2026-09-02, from the operator's usage page rather than the pricing
# page. The rates are real, but they did NOT replace the plan's quotas the way
# the announcement read: the account still meters a SESSION limit (3h reset)
# and a WEEKLY limit (4d reset), and paid "Extra usage" is a separate prepaid
# balance that starts at $0. So until someone tops that balance up, a sweep on
# this route spends quota and bills nothing -- which makes these figures
# NOTIONAL, exactly like the claude and codex ones, not an invoice.
#
# Keep the table anyway: notional or not, it is the only way to compare what a
# sweep costs across routes, and it is what caught the 9.4x money spread
# between glm-5.3-flash and glm-5.3 that identical token counts had hidden.
#
# And the kimi-k3 special case was NOT retired after all. Its library page
# warns it "requires a Pro or Max subscription, and consumes extra usage
# credits", and the usage page bears that out: 104 kimi-k3 requests consumed
# the whole session quota, while 579 minimax-m3 requests over the same week
# were a thin slice of it. Per REQUEST it is in a different class from
# everything else here, and the dollar column does not show that.
#
# DEEPSEEK HAS TWO TIERS since 2026-09-07 (ollama's mailing, confirmed on
# ollama.com/pricing the same day). The page now labels the halved figures
# "Standard" and the old ones "Peak": peak is 12:00-18:00 UTC Monday-Friday,
# everything else -- weekday mornings and evenings, all weekend -- is standard.
#   deepseek-v4-flash  standard 0.22 / 0.66 / 0.007   peak 0.44 / 1.32 / 0.014
#   deepseek-v4-pro    standard 0.66 / 1.98 / 0.022   peak 1.32 / 3.96 / 0.044
# The table carries STANDARD, for two reasons: it is ollama's own base label,
# and both measured deepseek sweeps ran in it -- pro on 2026-09-01 at 09:22
# UTC, flash on 2026-09-02 at 06:26 UTC (run ids in results.csv are local
# time, UTC+2). A sweep started in a European afternoon (14:00-20:00 local)
# would bill at peak, 2x these; read a deepseek dollar figure with that in mind,
# and check the run id before quoting one in a seat decision. Ollama says
# off-peak pricing "for more models will be available soon", so this note may
# grow into a general mechanism; it is not one yet.
OLLAMA_RATES = {
    # deepseek-v4.1-flash, added 2026-09-13 the day it was benched. A THIRD
    # deepseek tier, cheaper than v4-flash on every column and 7x cheaper on
    # cached input -- the model page credits its Causal Encoder-Decoder design,
    # which cuts the global KV cache to ~1/4 of v4-flash. Peak (12:00-18:00 UTC
    # Mon-Fri) is 0.30 / 1.20 / 0.006, the same 2x the other two deepseek rows
    # carry; the sweep that measured it ran Sunday 20:xx UTC, i.e. standard.
    "deepseek-v4.1-flash": (0.15, 0.60, 0.003),
    "deepseek-v4-flash": (0.22, 0.66, 0.007),
    "deepseek-v4-pro":   (0.66, 1.98, 0.022),
    "gemma4":            (0.14, 0.40, 0.05),
    "glm-5.1":           (1.00, 3.20, 0.20),
    "glm-5.2":           (1.40, 4.40, 0.26),
    "glm-5.3":           (1.40, 4.40, 0.26),
    "glm-5.3-flash":     (0.15, 0.50, 0.03),
    "gpt-oss:20b":       (0.07, 0.30, 0.035),
    "gpt-oss:120b":      (0.15, 0.60, 0.014),
    "kimi-k2.6":         (0.95, 4.00, 0.16),
    "kimi-k2.7-code":    (0.95, 4.00, 0.19),
    "kimi-k3":           (3.00, 15.00, 0.30),
    "minimax-m2.7":      (0.30, 1.20, 0.06),
    "minimax-m3":        (0.60, 2.40, 0.12),
    "mistral-large-3":   (0.50, 1.50, 0.50),
    "nemotron-3-nano":   (0.06, 0.24, 0.06),
    "nemotron-3-super":  (0.015, 0.60, 0.015),
    "nemotron-3-ultra":  (0.10, 3.00, 0.10),
    "qwen3.5:397b":      (0.60, 3.60, 0.60),
}


# WHEN PROMPT CACHING BECAME AVAILABLE ON EACH ROUTE.
#
# AVAILABLE is the word that matters, and it is NOT the same as "this run was
# cached". The era says what the route could do on the day a row was measured;
# whether a particular session actually read from cache is a separate question
# with a separate answer (see cached_share, and the warning in
# check_caching_eras). Getting these two confused is what the 2026-09-13
# probing round corrected.
#
# What the era IS good for: cached input is priced at a small fraction of fresh
# input -- 2% on ollama's deepseek rows -- so a route that could not cache at
# all billed every sweep at the fresh rate, and a dollar column from before the
# switch cannot be compared with one from after it.
#
# Each entry is (last date OBSERVED without caching available, first date
# OBSERVED with it), as YYYYMMDD. The window between them is deliberately left
# UNKNOWN rather than split at a guessed date: nothing was measured in it, and
# inventing a cutoff would put a false era label on any row that later lands
# there.
#
# ollama: every row measured through 2026-09-02 records cache_read=0 across all
# eight targets and every session, and the route discarded cache_control (the
# claim then in config/agents/deepseek-ollama.yaml). By 2026-09-13 caching was
# available -- probed with identical back-to-back calls, six of the thirteen
# models already in the grid collapsed their second call's input into a cache
# read (deepseek-v4-flash 48371 -> 804, deepseek-v4-pro 48368 -> 804, gemma4
# 45676 -> 12, glm-5.3 46978 -> 758, gpt-oss:120b 38558 -> 46, minimax-m3
# 45723 -> 1). Those models have 0 in their recorded rows, so the ROUTE changed;
# no candidate's own behaviour explains it.
#
# BUT A CACHE HIT IS NOT GUARANTEED, AND IS NOT A PROPERTY OF THE MODEL. The
# other seven models missed entirely in that round, and a second round minutes
# later flipped nemotron-3-ultra from two clean misses (51215, cache 0) to two
# clean hits (in 7278/8015, cache 43200 both times), while glm-5.3-flash --
# which HAD cached in an earlier round the same day, 0 -> 40320 -- missed three
# times running. kimi-k3 read 0, 0, then 871. So the hit rate varies between
# calls to one model within minutes, which fits a per-instance cache behind a
# fleet the caller does not choose. Treat cache_read as a property of the RUN,
# never of the candidate, and rank cost across rows by the uncached-equivalent
# column, which does not depend on how the routing fell.
#
# The claude, anthropic and openrouter routes have cached for as long as this
# bench has existed; they get no entry and are treated as always-available.
ROUTE_CACHING = {
    "ollama": ("20260902", "20260913"),
}


def caching_era(harness, dates):
    """"cached", "uncached" or "unknown" for a row, from the dates it was run.

    `dates` is every YYYYMMDD a candidate's runs carry. A sweep that straddles
    the switch is "unknown" too: its rows were not all billed the same way, so
    no single label is true of the total.
    """
    window = ROUTE_CACHING.get(harness)
    if window is None:
        return "cached"
    last_uncached, first_cached = window
    eras = {("uncached" if d <= last_uncached else
             "cached" if d >= first_cached else "unknown") for d in dates}
    return eras.pop() if len(eras) == 1 else "unknown"


def check_caching_eras(rows):
    """Fail when a measured row contradicts ROUTE_CACHING; warn where it only might.

    Only ONE direction is decidable. An uncached-era row that recorded cached
    tokens is a contradiction -- caching was not available then, so the table's
    date is wrong -- and that is fatal. The reverse is not: a cached-era sweep
    that read nothing may simply never have hit a warm instance, because hits
    are not guaranteed (see ROUTE_CACHING), so it only warns.
    """
    bad = []
    per_model = {}
    for r in rows:
        harness = route(r["model"])
        if harness not in ROUTE_CACHING:
            continue
        date = r["run"].split("-")[0]
        if caching_era(harness, [date]) == "uncached" and int(r["cache_read"]):
            bad.append(f"  {r['run']} {r['model']} {r.get('target', r['task'])}: "
                       f"cache_read={r['cache_read']} in the uncached era")
        acc = per_model.setdefault(r["model"], {"dates": set(), "cache": 0, "sessions": 0,
                                                "harness": harness})
        acc["dates"].add(date)
        acc["cache"] += int(r["cache_read"])
        acc["sessions"] += int(r["sessions"])
    # A candidate labelled cached-era whose WHOLE sweep read nothing from cache.
    # This was a hard error in the first version of this function and is now only
    # a warning, because the assumption under it turned out to be false: a cache
    # hit is not guaranteed even when caching is available (see ROUTE_CACHING).
    # glm-5.3-flash missed three times running on a day it had hit, so a sweep
    # CAN legitimately come back all-zero and failing the report over it would be
    # a false alarm. It is still worth saying out loud -- it is equally what a
    # too-early date in the table looks like, and the two are told apart by
    # re-probing the route, not by reading this file.
    for model, acc in sorted(per_model.items()):
        if (caching_era(acc["harness"], sorted(acc["dates"])) == "cached"
                and acc["cache"] == 0 and acc["sessions"] >= 8):
            print(f"report.py: note: {model} ran {acc['sessions']} sessions with "
                  f"caching available on {acc['harness']} and read 0 cached tokens. "
                  f"Expected if the routing simply never hit; also what a "
                  f"too-early ROUTE_CACHING date looks like. Re-probe to tell "
                  f"them apart.", file=sys.stderr)
    if bad:
        sys.exit("report.py: measured rows contradict ROUTE_CACHING. Its dates are "
                 "wrong, or the route changed again -- fix the table rather than "
                 "the rows, and re-probe the route before trusting a cost column:\n"
                 + "\n".join(bad))


def cached_share(tok_in, cache_read):
    """Share of a sweep's INPUT that was served from cache, 0.0-1.0."""
    total = tok_in + cache_read
    return (cache_read / total) if total else 0.0


def run_cost_uncached(model, harness, tok_in, tok_out, cache_read, asof=None):
    """What the sweep would have cost with every input token billed fresh.

    The era-NEUTRAL figure: it is what an uncached-era row already paid, and the
    counterfactual for a cached-era one, so the two can be compared directly.
    Use it to rank cost across the switch; use run_cost() for what a sweep
    actually costs today.
    """
    return run_cost(model, harness, tok_in + cache_read, tok_out, 0, asof)


# The run id shape bench/run.sh records: fixpoint's run directory name.
RUN_ID = re.compile(r"[0-9]{8}-[0-9]{6}")


def ollama_rate(model):
    """Rates for an ollama tag, whose name carries a tag the price list does not.

    `glm-5.3:cloud` is priced as `glm-5.3`, `deepseek-v4-flash:0731-cloud` as
    `deepseek-v4-flash`, and `gemma4:31b-cloud` as `gemma4` -- but `gpt-oss:120b`
    and `qwen3.5:397b` ARE priced per size, so the size tag cannot simply be
    stripped. Try the full name, then the name with `-cloud` removed, then the
    stem.
    """
    cands = [model]
    if ":" in model:
        stem, tag = model.split(":", 1)
        if tag.endswith("-cloud"):
            cands.append(f"{stem}:{tag[:-len('-cloud')]}")
        cands.append(stem)
    for c in cands:
        if c in OLLAMA_RATES:
            return OLLAMA_RATES[c]
    return None


# What an AGENT ran BEFORE its yaml was last changed, as
# {agent: [(changed_on, model, effort), ...]} with the list in date order.
#
# THIS EXISTS BECAUSE AN AGENT NAME IS NOT A MEASUREMENT (found 2026-09-22, by
# swapping the claude seat). `claude` and `codex` resolve through their yaml at
# RENDER time, while results.csv only ever records the agent name -- so the day
# the model line changes, every older row for that agent silently re-labels AND
# RE-PRICES itself as the new model. Swapping claude-opus-5 -> claude-opus-5-5
# restated the August `claude` baseline from ~$8.61 to ~$6.55 (~$0.023 ->
# ~$0.017/pt): the same 376/484 sweep, billed at rates its tokens never paid,
# on a table that is published. Nothing in the recorded data can catch it --
# bench/summaries/*.json stores `"model": "claude"` and no resolved id -- so the
# history has to be written down here when the yaml changes.
#
# A row resolves to the FIRST entry whose changed_on is after the row's last
# measured date; rows on or after the final change resolve through the yaml.
# Dates carry a TIME on purpose. A calendar-day comparison is one granule too
# coarse in the one case that matters: a run measured on the change day but
# BEFORE the edit would fail a `day <` test and be priced at the new model's
# rates though its tokens ran on the old one -- the very restatement this table
# exists to prevent, just narrower. And a same-day re-measure is not a remote
# possibility: changing a seat is exactly when someone re-benches the alias.
# `2026-09-22 23:27:01` is when commit 1620c5f changed config/agents/claude.yaml.
AGENT_MODEL_HISTORY = {
    "claude": [("2026-09-22 23:27:01", "claude-opus-5", "high")],
}


def _stamp(s):
    """Any of a run id, a day, or a dated time -> one sortable YYYYMMDDHHMMSS.

    Accepts `20260922-230101` (run id, the live path), `20260922`, `2026-09-22`
    and `2026-09-22 23:27:01`. A value with no time means the START of that day,
    so a bare day compares as "before anything that happened during it".
    """
    digits = re.sub(r"\D", "", s)
    if len(digits) < 8:
        raise ValueError(f"not a date: {s!r}")
    return (digits[:14] + "0" * 14)[:14]


def _asof(e):
    """The date to resolve an aggregated entry's agent at: its LAST measured run.

    Last rather than first, and SAFE ONLY BECAUSE THE GROUP IS SINGLE-EPOCH by
    the time this is called -- the aggregation drops an alias's older epoch as
    soon as it has a newer one. Without that, "latest date" would be the bug
    rather than the rule: an entry sums tokens across every run it contains, so
    a group straddling a seat change would bill August's tokens at September's
    rates, which is the restatement AGENT_MODEL_HISTORY exists to prevent.

    An earlier version of this docstring claimed re-measuring made a pooled row
    "converge" to the new model. It did not; it averaged the two. The dropping
    is what makes the claim true, so the two must stay together.
    """
    return max(e.get("runs") or e.get("dates") or [None])


def agent_model(model, asof=None):
    """The model id behind a candidate name, for pricing and for display.

    `claude` and `codex` are agent names: what they run is whatever their yaml
    says today, so a row that only says "claude" stops being a measurement the
    moment that file changes. Pass `asof` -- the row's last measured date, as
    YYYYMMDD-HHMMSS or YYYY-MM-DD -- to resolve it as of when it was MEASURED
    rather than as of today. See AGENT_MODEL_HISTORY.
    """
    path = AGENTS / f"{model}.yaml"
    if not path.exists():
        return None, None
    if asof is not None:
        when = _stamp(asof)
        for changed_on, was_model, was_effort in AGENT_MODEL_HISTORY.get(model, ()):
            if when < _stamp(changed_on):
                return was_model, was_effort
    name, effort = None, None
    for line in path.read_text().splitlines():
        if line.startswith("model:"):
            name = line.split(":", 1)[1].split("#")[0].strip()
        elif line.startswith("effort:"):
            effort = line.split(":", 1)[1].split("#")[0].strip()
    return name, effort


def resolve(model, asof=None):
    """Display name for the model actually measured."""
    name, effort = agent_model(model, asof)
    if name is None:
        return model
    return f"{name} ({effort})" if effort else name


def route(model):
    """Which harness and provider the bench used, mirroring bench/run.sh.

    It matters for reading the token column, which counts NON-CACHED tokens: a
    route that caches shows a smaller number for the same work, so two rows with
    the same token count are not the same spend.

    THE OLLAMA ROUTE USED TO BE THE SIMPLE CASE AND NO LONGER IS. Every ollama
    row measured through 2026-09-02 has cache_read=0 -- the route discarded
    cache_control, so its input was paid fresh every session. That stopped being
    true by 2026-09-13: probed with two identical back-to-back calls per model,
    the second call's cacheReadInputTokens read 46413 for deepseek-v4-flash,
    40320 for glm-5.3-flash and 45393 for minimax-m3, all of which have 0 in
    their recorded rows. So the ollama rows now split into two eras, and the
    older ones overstate what the same sweep would cost today -- deepseek-v4.1-
    flash's ~$0.35 against glm-5.3-flash's ~$0.78 is mostly that, not the model.
    Making them comparable needs a real mechanism (a per-run route-capability
    flag, like the peak/off-peak one deepseek already wants); until then the
    caveat lives here and in bench/models.txt.
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


def runs_codex(model):
    """True when this candidate's agent file invokes the codex CLI."""
    path = AGENTS / f"{model}.yaml"
    if not path.exists():
        return False
    for line in path.read_text().splitlines():
        # The command list's first entry, as `  - codex`; `command: [codex, ...]`
        # is not a form any agent file in this bundle uses.
        if line.strip() in ("- codex", "- codex.cmd"):
            return True
    return False


def cached_is_subset(model):
    """True where the harness reports cached input INSIDE its input total.

    The codex CLI does (OpenAI usage shape), on every route it drives: the
    bundled codex-harness agents and the codex@ OpenRouter template alike.
    """
    return runs_codex(model) or model.startswith("codex@")


def billing(model, harness):
    if harness.startswith("openrouter"):
        return "credits"
    if harness == "ollama":
        # CORRECTED 2026-09-02 from the operator's own usage page. The
        # 2026-09-01 move to published per-token rates did NOT remove the
        # plan's quotas: the page still shows a SESSION limit (3h reset) and a
        # WEEKLY limit (4d reset), and "Extra usage" is a separate prepaid
        # balance that starts at $0. So a sweep spends quota, not money, until
        # someone tops that balance up and the quotas are exhausted.
        #
        # This route is therefore a subscription like claude and codex, and its
        # dollar figure is notional for the same reason. The per-token rates
        # stay useful -- they are the only way to compare what a sweep COST
        # across routes -- but reporting them as money spent was wrong.
        return "sub (ollama)"
    # Which SUBSCRIPTION pays is a property of the harness, not of the agent's
    # name: `codex` was the only ChatGPT-auth candidate until gpt-6-astra was
    # added on 2026-09-05, and a name check reported that row as billing the
    # claude plan. Read the invocation instead, so any future candidate wired
    # through the codex harness is classified by what it actually runs.
    if runs_codex(model):
        return "sub (chatgpt)"
    return "sub (claude)"


def run_cost(model, harness, tok_in, tok_out, cache_read, asof=None):
    """What the whole sweep cost, in money, or None where no rate is published.

    Split out of cost_per_point so the total and the per-point figure cannot
    disagree: both are this one calculation. Cached input is billed at the
    cache rate where the route publishes one, and at the full input rate where
    it does not (grok and qwen3.8-max on OpenRouter, qwen3.5 and the nemotrons
    on ollama, whose cached and fresh rates are identical anyway).
    """
    name, _ = agent_model(model, asof)
    rates = RATES.get(name or model) or ollama_rate(model)
    if rates is None:
        return None
    rate_in, rate_out, rate_cache = rates
    return (tok_in * rate_in
            + cache_read * (rate_in if rate_cache is None else rate_cache)
            + tok_out * rate_out) / 1e6


def money(harness, usd):
    """Format money, marking a subscription figure as notional with a leading ~."""
    if usd is None:
        return "-"
    # openrouter bills a card per token; every other route here is a plan.
    notional = not harness.startswith("openrouter")
    return f"{'~' if notional else ''}${usd:,.2f}"


def cost_per_point(model, harness, tok_in, tok_out, cache_read, points, asof=None):
    """What one seeded defect cost, in the currency that route actually spends.

    A dollar figure wherever a published rate exists, prefixed with ~ when the
    route is a plan and the money is therefore notional (nothing is billed per
    token there); bare only for OpenRouter, which actually charges a card.
    ollama is a plan too -- it has published per-token rates since 2026-09-01
    but still meters a session and a weekly quota, with paid "Extra usage" as a
    separate opt-in balance -- so its figure is notional as well. The
    thousands-of-tokens fallback remains only for a model with no published
    rate at all.
    """
    if not points:
        return "-"
    usd = run_cost(model, harness, tok_in, tok_out, cache_read, asof)
    if usd is None:
        # No published rate anywhere: fall back to the quota-shaped figure.
        return f"{(tok_in + tok_out) / 1000 / points:.1f}k"
    # Same rule as money(): bare only where a card is charged per token.
    prefix = "" if harness.startswith("openrouter") else "~"
    return f"{prefix}${usd / points:.3f}"


def cached_cell(e):
    """How much of THIS SWEEP's input came from cache. A run fact, not a model fact.

    Do not read this column as a property of the candidate. Measured 2026-09-13
    on the ollama route: cache hits vary between back-to-back calls to one model
    within minutes -- nemotron-3-ultra missed twice and then hit twice at 43200,
    glm-5.3-flash hit once and missed three times the same day -- which fits a
    per-instance cache behind a fleet the caller does not choose. Two sweeps of
    the same model can therefore show very different numbers here, and neither
    is wrong. Rank cost by the uncached-equivalent column instead.

    `-` is not 0%: a row from before its route had caching available had no
    cache to miss, which is why its dollar column is not comparable with a later
    one. `?` marks a sweep that straddled the switch.
    """
    if e["era"] == "uncached":
        return "-"
    if e["era"] == "unknown":
        return "?"
    return f"{e['cached_share'] * 100:.0f}%"


def per_point_uncached(model, harness, e):
    """Cost per point with every input token billed fresh: comparable across eras.

    Identical to the per-point column for an uncached-era row, and the
    counterfactual for a cached one, so this is the column to rank cost by when
    the field spans a route change.
    """
    if not e["points"]:
        return "-"
    usd = run_cost_uncached(model, harness, e["tok_in"], e["tok_out"], e["cache_read"], _asof(e))
    if usd is None:
        return "-"
    prefix = "" if harness.startswith("openrouter") else "~"
    return f"{prefix}${usd / e['points']:.3f}"


def _cost_key(model, harness, e):
    """A sortable key for the per-point column, whose units are not comparable.

    Some rows are priced in dollars and some in thousands of tokens, because
    ollama sells quota rather than tokens. Sorting the raw numbers would
    interleave $0.137 with 16.1k meaninglessly, so token-priced rows are pushed
    above every dollar-priced one and sorted among themselves. The column then
    reads as two ordered groups rather than one nonsensical sequence.
    """
    raw = cost_per_point(model, harness, e["tok_in"], e["tok_out"],
                         e["cache_read"], e["points"], _asof(e))
    if raw == "-":
        return -1
    if raw.endswith("k"):
        return 1e6 + float(raw[:-1])
    return float(raw.lstrip("~$"))


def measured_on(dates):
    """When this model's rows were produced, from the run ids.

    A model's rows can span days -- claude's go and design are from the
    2026-08-20 sweep while its language rows are from 2026-09-01 -- and the gap
    matters, because a row is only as current as the manifest it was scored
    against. A range is shown rather than just the latest, so a stale half is
    visible instead of hidden behind a fresh date.
    """
    if not dates:
        return "-"
    def d(x):
        return f"{x[0:4]}-{x[4:6]}-{x[6:8]}"
    lo, hi = min(dates), max(dates)
    return d(hi) if lo == hi else f"{d(lo)[5:]}\u2026{d(hi)}"



PAGE_JS = """
// Click a header to sort. Values come from data-sort on each cell rather than
// from the rendered text, because the display strings are not sortable: scores
// read "376/484", tokens carry thousands separators, and the per-point column
// mixes dollars with token counts.
(function () {
  function val(row, i) {
    var c = row.children[i];
    if (!c) return "";
    var d = c.getAttribute("data-sort");
    return d === null ? c.textContent.trim() : d;
  }
  function renumber(tb) {
    var n = 1;
    Array.prototype.forEach.call(tb.rows, function (r) {
      var c = r.querySelector("td.rank");
      if (c) c.textContent = n++;
    });
  }
  Array.prototype.forEach.call(document.querySelectorAll("table"), function (t) {
    var head = t.tHead, body = t.tBodies[0];
    if (!head || !body) return;
    Array.prototype.forEach.call(head.rows[0].cells, function (th, i) {
      if (th.classList.contains("norank")) return;
      th.classList.add("sortable");
      th.addEventListener("click", function () {
        var rows = Array.prototype.slice.call(body.rows);
        // numeric when every non-empty value parses as a number
        var numeric = rows.every(function (r) {
          var v = val(r, i);
          return v === "" || v === "-" || !isNaN(parseFloat(v));
        });
        var desc = th.getAttribute("data-dir") !== "desc";
        rows.sort(function (a, b) {
          var x = val(a, i), y = val(b, i);
          if (numeric) {
            var nx = parseFloat(x), ny = parseFloat(y);
            if (isNaN(nx)) nx = -Infinity;
            if (isNaN(ny)) ny = -Infinity;
            return desc ? ny - nx : nx - ny;
          }
          return desc ? y.localeCompare(x) : x.localeCompare(y);
        });
        rows.forEach(function (r) { body.appendChild(r); });
        Array.prototype.forEach.call(head.rows[0].cells, function (o) {
          o.removeAttribute("data-dir");
          o.classList.remove("sorted");
        });
        th.setAttribute("data-dir", desc ? "desc" : "asc");
        th.classList.add("sorted");
        renumber(body);
      });
    });
  });
})();
"""

PAGE_CSS = """
:root {
  color-scheme: light dark;
  --ground: #F2F3F5;
  --surface: #FFFFFF;
  --ink: #151920;
  --muted: #5B6472;
  --rule: #DCDFE4;
  --rule-soft: #E9EBEF;
  --signal: #B87514;
  --signal-fill: rgba(184, 117, 20, 0.16);
  --critical: #A32B21;
  --critical-fill: rgba(163, 43, 33, 0.10);
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --ground: #101318;
    --surface: #171B22;
    --ink: #E7E9ED;
    --muted: #98A1AE;
    --rule: #262B34;
    --rule-soft: #1E232B;
    --signal: #E39B2E;
    --signal-fill: rgba(227, 155, 46, 0.18);
    --critical: #E2685C;
    --critical-fill: rgba(226, 104, 92, 0.12);
  }
}
:root[data-theme="dark"] {
  --ground: #101318;
  --surface: #171B22;
  --ink: #E7E9ED;
  --muted: #98A1AE;
  --rule: #262B34;
  --rule-soft: #1E232B;
  --signal: #E39B2E;
  --signal-fill: rgba(227, 155, 46, 0.18);
  --critical: #E2685C;
  --critical-fill: rgba(226, 104, 92, 0.12);
}

body {
  margin: 0;
  background: var(--ground);
  color: var(--ink);
  font-family: "IBM Plex Sans", system-ui, -apple-system, sans-serif;
  font-size: 15px;
  line-height: 1.55;
  -webkit-font-smoothing: antialiased;
}
.page {
  max-width: 1180px;
  margin: 0 auto;
  padding: 56px 28px 72px;
  display: flex;
  flex-direction: column;
  gap: 34px;
}
.eyebrow {
  font-family: "IBM Plex Sans Condensed", "IBM Plex Sans", sans-serif;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--signal);
}
h1 {
  font-family: "IBM Plex Sans Condensed", "IBM Plex Sans", sans-serif;
  font-size: clamp(30px, 4.4vw, 46px);
  font-weight: 600;
  letter-spacing: -0.015em;
  line-height: 1.08;
  text-wrap: balance;
  margin: 8px 0 0;
}
.lede {
  max-width: 62ch;
  color: var(--muted);
  margin: 14px 0 0;
}
.lede strong { color: var(--ink); font-weight: 600; }

.facts {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
  border-top: 1px solid var(--rule);
  border-bottom: 1px solid var(--rule);
}
.fact {
  flex: 1 1 150px;
  padding: 16px 22px 16px 0;
  border-right: 1px solid var(--rule-soft);
}
.fact:last-child { border-right: 0; }
.fact dt {
  font-family: "IBM Plex Sans Condensed", sans-serif;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--muted);
}
.fact dd {
  margin: 4px 0 0;
  font-family: "IBM Plex Mono", ui-monospace, monospace;
  font-size: 24px;
  font-variant-numeric: tabular-nums;
}

.scroller { overflow-x: auto; }
table { border-collapse: collapse; width: 100%; min-width: 940px; }
caption { text-align: left; color: var(--muted); font-size: 13px; padding-bottom: 12px; }
th {
  font-family: "IBM Plex Sans Condensed", sans-serif;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--muted);
  text-align: right;
  padding: 0 14px 10px 0;
  border-bottom: 1px solid var(--rule);
  white-space: nowrap;
}
th.left, td.left { text-align: left; }
td {
  padding: 11px 14px 11px 0;
  border-bottom: 1px solid var(--rule-soft);
  text-align: right;
  font-family: "IBM Plex Mono", ui-monospace, monospace;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
tbody tr:hover { background: var(--rule-soft); }
td.rank { color: var(--muted); width: 34px; padding-left: 10px; }
tr.failed td.rank { box-shadow: inset 3px 0 0 var(--critical); }
tr.failed { background: var(--critical-fill); }
.name { font-family: "IBM Plex Sans", sans-serif; font-size: 14px; font-weight: 600; }
.sub { color: var(--muted); font-size: 12px; font-family: "IBM Plex Mono", monospace; }
.tag {
  display: inline-block;
  margin-left: 7px;
  padding: 1px 6px;
  border: 1px solid var(--rule);
  border-radius: 2px;
  font-family: "IBM Plex Sans Condensed", sans-serif;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--muted);
  vertical-align: 1px;
}
.score { position: relative; width: 190px; padding-right: 14px; }
.bar {
  position: absolute;
  left: 0; top: 50%;
  transform: translateY(-50%);
  height: 22px;
  background: var(--signal-fill);
  border-left: 2px solid var(--signal);
}
.score span { position: relative; font-size: 14px; font-weight: 600; }
.score .of { color: var(--muted); font-weight: 400; font-size: 12px; }
.err { color: var(--critical); font-weight: 600; }
th.sortable { cursor: pointer; user-select: none; }
th.sortable:hover { color: var(--ink); }
th.sortable::after { content: "\\2195"; opacity: 0.35; margin-left: 5px; font-size: 10px; }
th.sorted::after { content: "\\25BC"; opacity: 1; color: var(--signal); }
th.sorted[data-dir="asc"]::after { content: "\\25B2"; }
.pct { font-weight: 600; }
tfoot td { border-top: 1px solid var(--rule); border-bottom: 0; padding-top: 12px; color: var(--muted); }
.zero { color: var(--muted); }
.notes {
  border-top: 1px solid var(--rule);
  padding-top: 20px;
  display: grid;
  gap: 10px;
  max-width: 78ch;
  font-size: 13px;
  color: var(--muted);
}
.notes b { color: var(--ink); font-weight: 600; }
code {
  font-family: "IBM Plex Mono", monospace;
  font-size: 0.92em;
  background: var(--rule-soft);
  padding: 1px 4px;
  border-radius: 2px;
}
@media (max-width: 640px) { .page { padding: 36px 18px 56px; } }
"""


def _esc(text):
    return (str(text).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;"))


def write_html(ranked, all_targets, out_path):
    """Emit the standings as one self-contained page, for sharing."""
    top = max((max(e["points"] for e in entries) for _, entries in ranked), default=100)
    # Counted, not assumed: 5 sessions per repeat is the protocol, but an
    # INCOMPLETE candidate ran only one task, and repeats multiply. The csv
    # records what actually ran, so the header cannot drift from the rows.
    sessions = sum(e["sessions"] for _, entries in ranked for e in entries)
    repeats = max((len(entries) for _, entries in ranked), default=1)
    # The measurement window, from the run ids. It was hardcoded to 2026-08-20
    # until the first row landed on another day, which is exactly the drift the
    # page exists to avoid -- it is generated so that it cannot disagree with
    # results.csv, and a stamped date is part of what it must not disagree with.
    days = sorted({d for _, entries in ranked for e in entries for d in e["dates"]})
    def _day(d):
        return f"{d[0:4]}-{d[4:6]}-{d[6:8]}"
    when = _day(days[0]) if len(days) < 2 else f"{_day(days[0])} &ndash; {_day(days[-1])}"
    failures = 0
    rows_html = []
    for i, (model, entries) in enumerate(ranked, 1):
        e = entries[0]
        # The Err column is failed SESSIONS per candidate, so the tile sums it;
        # it used to count candidates with any failure and disagreed with the
        # column it sits above (review run 20260907-150029).
        failures += e["errors"]
        mtok = e["tokens"] / 1e6
        eff = e["points"] / mtok if mtok else 0.0
        harness = route(model)
        measured = resolve(model, _asof(e))
        baseline = model in ("claude", "codex")
        width = 100.0 * e["points"] / max(top, 1)
        # "code only" was accurate when the scale was go+design; now a row can
        # be short of any of the eight, so name what is actually missing.
        incomplete = ("  (no " + ", ".join(e["missing"]) + ")"
                      if e["missing"] else "")
        rows_html.append(f"""      <tr class="{'failed' if e['errors'] else ''}">
        <td class="rank">{i}</td>
        <td class="left">
          <div class="name">{_esc(model)}{'<span class="tag">baseline</span>' if baseline else ''}</div>
          <div class="sub">{_esc(measured) if measured != model else ''}</div>
        </td>
        <td class="left sub">{_esc(harness)}</td>
        <td class="left sub">{_esc(billing(model, harness))}</td>
        <td class="score" data-sort="{e['points']}">
          <div class="bar" style="width: {width:.1f}%"></div>
          <span>{e['points']}<span class="of">/{e['possible']}{incomplete}</span></span>
        </td>
        <td data-sort="{e['tokens']}">{e['tokens']:,}</td>
        <td data-sort="{eff:.4f}">{eff:.0f}</td>
        <td data-sort="{run_cost(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], _asof(e)) or -1:.4f}">{_esc(money(harness, run_cost(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], _asof(e))))}</td>
        <td data-sort="{_cost_key(model, harness, e)}">{_esc(cost_per_point(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], e['points'], _asof(e)))}</td>
        <td class="{'zero' if e['era'] != 'cached' else ''}" data-sort="{-1 if e['era'] != 'cached' else e['cached_share']}">{_esc(cached_cell(e))}</td>
        <td data-sort="{(run_cost_uncached(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], _asof(e)) or 0) / (e['points'] or 1)}">{_esc(per_point_uncached(model, harness, e))}</td>
        <td data-sort="{e['seconds']}">{e['seconds']:,}</td>
        <td class="{'err' if e['errors'] else 'zero'}" data-sort="{e['errors']}">{e['errors']}</td>
        <td class="sub" data-sort="{max(e['dates']) if e['dates'] else ''}">{measured_on(e['dates'])}</td>
      </tr>""")

    # By-target matrix. Omitted while only one target exists, because a
    # single-column matrix says nothing the score column has not already said.
    matrix_html = ""
    if len(all_targets) > 1:
        cols = "".join(f'<th scope="col">{_esc(t)}</th>' for t in all_targets)
        body = []
        for model, entries in ranked:
            e = entries[0]
            cells = ""
            for t in all_targets:
                got = e["by_target"].get(t)
                if got is None:
                    cells += '<td class="zero" data-sort="-1">&mdash;</td>'
                else:
                    pct = got[0] / got[1] * 100
                    bad = len(got) > 2 and got[2]
                    cells += (f'<td{" class=\"err\"" if bad else ""} '
                              f'data-sort="{pct:.4f}">'
                              f'<span class="pct">{pct:.0f}%{"!" if bad else ""}</span>'
                              f'<span class="sub"> {got[0]}/{got[1]}</span></td>')
            body.append(f'      <tr><td class="left"><div class="name">'
                        f'{_esc(model)}</div></td>{cells}</tr>')
        foot = ""
        for t in all_targets:
            # Only clean cells: a `!` cell lost a lens and its recall is
            # deflated, and the note beneath promises it is not in the median.
            rates = [e[0] / e[1] for m, es in ranked
                     for e in [es[0]["by_target"].get(t)]
                     if e and not (len(e) > 2 and e[2])]
            foot += ('<td class="zero">&mdash;</td>' if not rates else
                     f'<td><span class="pct">'
                     f'{statistics.median(rates) * 100:.0f}%</span></td>')
        rows = "\n".join(body)
        matrix_html = f"""  <div class="scroller">
    <table>
      <caption>Recall per target. The score above collapses these into one
      number, which is what hides a target the whole field is weak on.</caption>
      <thead>
        <tr><th class="left" scope="col">Candidate</th>{cols}</tr>
      </thead>
      <tbody>
{rows}
      </tbody>
      <tfoot>
        <tr><td class="left"><div class="name">field median</div></td>{foot}</tr>
      </tfoot>
    </table>
  </div>"""

    page = f"""<title>Fixpoint Reviewer Bench</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;600&family=IBM+Plex+Sans+Condensed:wght@600&family=IBM+Plex+Sans:wght@400;600&display=swap">
<style>{PAGE_CSS}</style>
<div class="page">
  <header>
    <div class="eyebrow">Reviewer benchmark &middot; {when}</div>
    <h1>Which model is worth a panel seat</h1>
    <p class="lede">Every candidate reviews the same frozen targets &mdash; the same
    link-shortener service in <strong>{len(all_targets) - 1} languages</strong>, plus a design
    document &mdash; and is scored against planted defects. No judge, no taste: a deterministic
    matcher credits each finding to at most one seed. Nobody scores full marks, and that is the
    point. What matters is the distance to the <strong>claude baseline</strong>, and what the
    points cost. The headline score is <strong>every target</strong>, because the go+design
    pair it used to be misranked six of these models by three places or more once the other
    six languages disagreed with it &mdash; the per-language matrix below is where a model
    that is strong in one language and weak in another becomes visible.</p>
  </header>

  <dl class="facts">
    <div class="fact"><dt>Candidates</dt><dd>{len(ranked)}</dd></div>
    <div class="fact"><dt>Sessions</dt><dd>{sessions}</dd></div>
    <div class="fact"><dt>Targets</dt><dd>{len(all_targets)}</dd></div>
    <div class="fact"><dt>Repeats</dt><dd>{repeats}</dd></div>
    <div class="fact"><dt>Contract failures</dt><dd>{failures}</dd></div>
  </dl>

  <div class="scroller">
    <table>
      <caption>Ranked by score. One repeat each &mdash; single-run recall is a sample, not a verdict.</caption>
      <thead>
        <tr>
          <th class="left norank" scope="col">#</th>
          <th class="left" scope="col">Candidate</th>
          <th class="left" scope="col">Route</th>
          <th class="left" scope="col">Pays</th>
          <th class="left" scope="col">Score</th>
          <th scope="col">Tokens</th>
          <th scope="col">Pts / Mtok</th>
          <th scope="col">Total $</th>
          <th scope="col">Per point</th>
          <th scope="col">Cached (this run)</th>
          <th scope="col">Per point (uncached-eq)</th>
          <th scope="col">Wall (s)</th>
          <th scope="col">Err</th>
          <th scope="col">Measured</th>
        </tr>
      </thead>
      <tbody>
{chr(10).join(rows_html)}
      </tbody>
    </table>
  </div>

  {matrix_html}

  <div class="notes">
    <p><b>Per point</b> is what one found defect cost. <code>$</code> is money actually
    billed to a card, which is OpenRouter and only OpenRouter. <code>~$</code> is notional:
    the route bills a plan rather than the tokens, so the figure is what the same run would
    have cost at published rates &mdash; the only way to compare a seat paid by plan against
    one paid in credits. That covers Claude, ChatGPT <em>and</em> ollama: ollama published
    per-token rates on 2026-09-01, but it still meters a session quota and a weekly quota,
    and its paid &ldquo;Extra usage&rdquo; is a separate balance that starts empty. A sweep
    there spends quota, not money. Every route in this table is priced in the same unit;
    only one of them is an invoice.</p>
    <p><b>Cached (this run)</b> is a fact about the sweep, not about the model.
    Prompt caching became available on the ollama route between 2026-09-02 and
    2026-09-13, and cached input there costs about 2% of fresh &mdash; but a hit is
    not guaranteed. Probing thirteen models the same day, six collapsed a repeated
    call into a cache read and seven did not, and minutes later nemotron-3-ultra had
    flipped from missing to hitting while glm-5.3-flash, which had hit earlier that
    day, missed three times running. That fits a per-instance cache behind a fleet
    the caller does not choose. So two sweeps of one model can show very different
    numbers in this column and neither is wrong, and a <code>&ndash;</code> is not
    zero: it marks a run from before caching existed on that route, which had no
    cache to miss.</p>
    <p>Which is why there are two cost columns. <b>Per point</b> is what the sweep
    cost as billed on the day it ran, routing luck included. <b>Per point
    (uncached-eq)</b> prices every input token fresh: what a pre-caching row already
    paid, the counterfactual for a later one, and the only figure here that does not
    move with how the routing fell &mdash; so it is the column to rank cost by. It is
    also what separates a real saving from an artefact: the ollama rows that look
    three times cheaper than their neighbours read level on it. <em>Scores are not
    affected</em> &mdash; caching changes what a run costs, not what it finds.</p>
    <p><b>Contract failure</b> is the disqualifier, independent of score: a session that
    returns unparseable output burns a full slot and can flip a panel verdict to inconclusive.
    Both models that failed here have failed before.</p>
    <p><b>Wall clock</b> matters as much as tokens: a panel round runs at the pace of its
    slowest reviewer.</p>
    <p><b>!</b> marks a cell whose run lost a lens to a contract failure. That recall is
    <em>deflated, not low</em> &mdash; the findings the failed lens would have contributed were
    never parsed &mdash; so it is not comparable to a clean cell and is excluded from the field
    median. Re-run the target to get a real number; the failure itself still counts against the
    model.</p>
  </div>
</div>
<script>{PAGE_JS}</script>
"""
    pathlib.Path(out_path).write_text(page)
    print(f"wrote {out_path}")


def seed_value(results_dir):
    """Per-seed hit rate across every scored run, the retire/replace instrument.

    A panel seat is decided by measurement; a SEED is worth keeping by the same
    standard. Reads the per-run tables score.py already writes to results/ and,
    for each seed, reports the fraction of runs that found it. A seed nearly
    everyone finds carries almost no information about the model under test --
    it costs tokens and never moves a score -- and a seed nobody finds is either
    broken or beyond the whole field; both are candidates to retire. The
    discriminating middle is what the bench is actually made of.

    Denominator is runs-per-target, not models: repeats count, because a seed a
    model finds once and misses once is exactly the kind of low-signal seed this
    is meant to surface. With every row at one repeat today the two are equal.
    """
    RETIRE, DEAD = 0.90, 0.0
    # Below this many runs a hit rate is not a measurement: with one run every
    # found seed reads 100% and every miss 0%, which would flag most of a fresh
    # target for retirement on a sample of one.
    MIN_RUNS = 5
    tasks = {}
    rx = re.compile(r"^\|\s*([A-Z]+\d+)\s*\|\s*(YES|—)\s*\|", re.M)
    for f in sorted(pathlib.Path(results_dir).glob("*.md")):
        # {model}-{target}-{runid}.md, and a model name may itself contain
        # dashes -- so the target is the token immediately before the run id,
        # not a fixed alternation. Anchoring on the run id keeps this working
        # for every target added later without another edit here.
        m = re.match(r".+-([A-Za-z0-9_.+#]+)-\d{8}-\d{6}\.md$", f.name)
        if not m:
            continue
        task = m.group(1)
        seen = tasks.setdefault(task, {"runs": 0, "hit": {}, "order": []})
        seen["runs"] += 1
        for sid, verdict in rx.findall(f.read_text()):
            if sid not in seen["hit"]:
                seen["hit"][sid] = 0
                seen["order"].append(sid)
            if verdict == "YES":
                seen["hit"][sid] += 1
    if not tasks:
        sys.exit(f"{results_dir}: no per-run tables to read")

    # A retired seed is out of the denominator already; listing it as a retire
    # candidate would be noise, and its rate is still worth showing so the
    # decision stays auditable.
    root = pathlib.Path(results_dir).parent
    retired = {}
    for task in tasks:
        mf = root / f"manifest-{task}.yaml"
        if not mf.exists():
            continue
        cur = None
        for line in mf.read_text().splitlines():
            t = line.strip()
            if t.startswith("- id:"):
                cur = t.split(":", 1)[1].strip()
            elif cur and t.startswith("retired:") and "true" in t.lower():
                retired.setdefault(task, set()).add(cur)

    for task in sorted(tasks):
        d = tasks[task]
        n = d["runs"]
        out = retired.get(task, set())
        rows = sorted(((sid, d["hit"][sid] / n) for sid in d["order"]),
                      key=lambda kv: (-kv[1], kv[0]))
        scored = [(sid, r) for sid, r in rows if sid not in out]
        thin = n < MIN_RUNS
        retire = [sid for sid, r in scored if r >= RETIRE]
        dead = [sid for sid, r in scored if r <= DEAD]
        live = len(scored) - len(retire) - len(dead)
        head = (f"\n{task}: {len(scored)} scored seeds over {n} run(s)"
                + (f", {len(out)} already retired" if out else ""))
        if thin:
            print(head + f"  -- ONLY {n} RUN(S): rates below are not a measurement, "
                  f"no seed should be retired on them")
        else:
            print(head + f"  [{len(retire)} near-dead >={RETIRE:.0%}, "
                  f"{live} discriminating, {len(dead)} never found]")
        for sid, rate in rows:
            if sid in out:
                flag = "(gone)"
            elif thin:
                flag = "      "
            else:
                flag = "RETIRE" if rate >= RETIRE else ("DEAD  " if rate <= DEAD else "      ")
            bar = "#" * round(rate * 20)
            print(f"  {flag} {sid:5s} {rate*100:5.1f}%  {bar}")
    print("\nRETIRE = found by nearly every model, so it no longer separates them; "
          "replace with a harder defect.")
    print("DEAD   = found by nobody: broken anchor/keywords, or beyond the field. "
          "Check the manifest before assuming it is just hard.")
    return 0


README_BEGIN = "<!-- BENCH TABLE: generated by `make bench-readme`, do not edit -->"
README_END = "<!-- END BENCH TABLE -->"


def write_json(ranked, all_targets, out_path):
    """The standings as data, for anything that renders them somewhere else.

    www.saiko.cz/ai-benchmarks builds its page from this file, so the page and
    the README table are two renderings of one computation rather than two
    sources: every number here is the same call that produced the html and the
    markdown. Token counts are the normalised ones (see main), costs carry the
    notional flag the html marks with ~, and by_target keeps the deflation flag
    the `!` marker is drawn from.
    """
    def _day(d):
        return f"{d[0:4]}-{d[4:6]}-{d[6:8]}"
    pool = {}
    cands = []
    for i, (model, entries) in enumerate(ranked, 1):
        e = entries[0]
        harness = route(model)
        name, effort = agent_model(model, _asof(e))
        usd = run_cost(model, harness, e["tok_in"], e["tok_out"], e["cache_read"], _asof(e))
        uncached = run_cost_uncached(model, harness, e["tok_in"], e["tok_out"], e["cache_read"], _asof(e))
        mtok = e["tokens"] / 1e6
        for t, got in e["by_target"].items():
            pool.setdefault(t, got[1])
        cands.append({
            "rank": i,
            "candidate": model,
            # The html tags claude and codex as the calibration pair.
            "baseline": model in ("claude", "codex"),
            "model": name or model,
            "effort": effort,
            "route": harness,
            "pays": billing(model, harness),
            "points": e["points"],
            "possible": e["possible"],
            "incomplete": bool(e["missing"]),
            "tokens": e["tokens"],
            "points_per_mtok": round(e["points"] / mtok, 1) if mtok else None,
            "cost_usd": None if usd is None else round(usd, 2),
            "cost_per_point_usd": (None if usd is None or not e["points"]
                                   else round(usd / e["points"], 3)),
            "notional": not harness.startswith("openrouter"),
            # Which prompt-caching era the cost fields belong to. "uncached"
            # rows paid every input token fresh because their route did not
            # cache yet, so their cost_usd is NOT comparable with a "cached"
            # row's. cost_per_point_uncached_usd is the era-neutral figure --
            # identical to cost_per_point_usd for an uncached row, the
            # counterfactual for a cached one -- and is what a consumer of this
            # file should rank cost by.
            "caching_era": e["era"],
            "cached_input_share": (None if e["era"] != "cached"
                                   else round(e["cached_share"], 3)),
            "cost_per_point_uncached_usd": (
                None if uncached is None or not e["points"]
                else round(uncached / e["points"], 3)),
            "seconds": e["seconds"],
            "sessions": e["sessions"],
            "errors": e["errors"],
            "measured": {"first": _day(min(e["dates"])), "last": _day(max(e["dates"]))}
                        if e["dates"] else None,
            "by_target": {t: {"found": got[0], "of": got[1],
                              "deflated": bool(len(got) > 2 and got[2])}
                          for t, got in e["by_target"].items()},
        })
    days = sorted({d for _, entries in ranked for e in entries for d in e["dates"]})
    doc = {
        "source": "https://github.com/dsaiko/fixpoint/blob/main/bench/results.csv",
        "generated_by": "bench/report.py --json",
        "targets": all_targets,
        "pool": pool,
        "possible": sum(pool.get(t, 0) for t in SCORED_TARGETS if t in pool),
        "measured": {"first": _day(days[0]), "last": _day(days[-1])} if days else None,
        "totals": {
            "candidates": len(ranked),
            "sessions": sum(e["sessions"] for _, entries in ranked for e in entries),
            "targets": len(all_targets),
            "contract_failures": sum(entries[0]["errors"] for _, entries in ranked),
        },
        "candidates": cands,
    }
    pathlib.Path(out_path).write_text(json.dumps(doc, indent=1, ensure_ascii=False) + "\n")
    print(f"wrote {out_path}")


def check_markers(out_path):
    """The README's contents, after proving both table markers are present.

    Called once before any output is written and again by write_markdown, so
    the markdown target can never fail after the html has already gone out.
    """
    doc = pathlib.Path(out_path).read_text()
    if README_BEGIN not in doc or README_END not in doc:
        sys.exit(f"{out_path}: markers not found. Add these two lines where the "
                 f"table belongs:\n  {README_BEGIN}\n  {README_END}")
    return doc


def write_markdown(ranked, all_targets, out_path):
    """Rewrite the block between the markers in a file, in place.

    The table is GENERATED for the same reason the HTML page is: a hand-pasted
    copy silently stops matching results.csv the first time a model is added,
    and a stale leaderboard in the README is worse than none, because it is the
    one number people quote without opening the CSV. `make bench-readme`
    regenerates it; the markers are the contract.
    """
    lines = [README_BEGIN, ""]
    lines.append(f"{len(ranked)} candidates, scored on {len(all_targets)} targets "
                 f"({', '.join(all_targets)}).")
    lines.append("")
    lines.append("| # | candidate | route | score | recall | total $ | per point | cached (this run) | per point (uncached-eq) | wall s | fails |")
    lines.append("|--:|---|---|--:|--:|--:|--:|--:|--:|--:|--:|")
    for i, (model, entries) in enumerate(ranked, 1):
        e = entries[0]
        harness = route(model)
        pct = 100 * e["points"] / e["possible"] if e["possible"] else 0
        note = " *(incomplete)*" if e["missing"] else ""
        lines.append(
            f"| {i} | `{model}`{note} | {harness} | {e['points']}/{e['possible']} "
            f"| {pct:.0f}% "
            f"| {money(harness, run_cost(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], _asof(e)))} "
            f"| {cost_per_point(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], e['points'], _asof(e))} "
            f"| {cached_cell(e)} | {per_point_uncached(model, harness, e)} "
            f"| {e['seconds']} | {e['errors']} |")
    lines.append("")
    lines.append("Recall per target, which is where a model that is strong in one "
                 "language and weak in another shows up:")
    lines.append("")
    lines.append("| candidate | " + " | ".join(all_targets) + " |")
    lines.append("|---|" + "--:|" * len(all_targets))
    for model, entries in ranked:
        cells = []
        for tgt in all_targets:
            got = entries[0]["by_target"].get(tgt)
            if not got:
                cells.append("&mdash;")
            else:
                mark = "!" if len(got) > 2 and got[2] else ""
                cells.append(f"{got[0]}/{got[1]}{mark}")
        lines.append(f"| `{model}` | " + " | ".join(cells) + " |")
    lines.append("")
    lines.append("`!` marks a cell that lost a lens to a contract failure: its recall is "
                 "deflated, not low. `~$` is notional &mdash; the route bills a plan, not "
                 "the tokens (claude, codex and ollama, which still meters session and "
                 "weekly quotas); a bare `$` is money actually billed to a card, which is "
                 "OpenRouter only.")
    lines.append("")
    lines.append(README_END)
    block = "\n".join(lines)

    path = pathlib.Path(out_path)
    doc = check_markers(out_path)
    head, rest = doc.split(README_BEGIN, 1)
    _, tail = rest.split(README_END, 1)
    path.write_text(head + block + tail)
    print(f"{out_path}: bench table regenerated ({len(ranked)} candidates)")


def drop_stale_epochs(by_mt):
    """Keep only an alias's CURRENT model, in place, once it has been measured on one.

    See the comment at the call site for why. Split out so it can be tested:
    the aggregation around it is inline in main() and reachable only by writing
    a csv, while this is the whole of the rule and is pure.
    """
    for key, rs in by_mt.items():
        alias = key[0]
        if alias not in AGENT_MODEL_HISTORY:
            continue
        epochs = {r["run"]: agent_model(alias, r["run"])[0] for r in rs}
        if len(set(epochs.values())) > 1:
            current = epochs[max(epochs)]
            by_mt[key] = [r for r in rs if epochs[r["run"]] == current]
    return by_mt


def selftest():
    """Assert the agent-history pricing guard still guards. Run by `make bench-check`.

    This exists because the guard's only proof was once a one-time manual check
    that the regenerated table came out byte-identical -- which cannot be re-run
    on the day it matters, i.e. the next time a seat's yaml changes. Everything
    here is about that one mechanism; the rest of the module is covered by the
    fact that its output is generated and diffed.
    """
    fails = []

    def check(label, got, want):
        if got != want:
            fails.append(f"  {label}\n     got  {got!r}\n     want {want!r}")

    # 1. The regression itself: a `claude` row measured in August must resolve
    #    to the model that actually ran it, not to whatever the yaml says today.
    check("agent_model('claude', <august run id>)",
          agent_model("claude", "20260901-120000"), ("claude-opus-5", "high"))
    check("resolve('claude', <august run id>)",
          resolve("claude", "20260901-120000"), "claude-opus-5 (high)")


    # 3. Every shape _asof can hand over must mean the same instant. The run-id
    #    form is the live path; the others are what a future caller may pass.
    for shape in ("20260901-120000", "20260901", "2026-09-01", "2026-09-01 12:00:00"):
        check(f"date shape {shape!r} resolves like the run id",
              agent_model("claude", shape)[0], "claude-opus-5")
    check("_stamp pads a bare day to the START of it",
          _stamp("2026-09-22"), "20260922000000")

    # 3b. The precision that day-granularity got wrong: a run measured on the
    #     change DAY but before the edit must still price at the OLD model.
    check("same-day run BEFORE the change keeps the old model",
          agent_model("claude", "20260922-230101")[0], "claude-opus-5")
    check("same-day run AFTER the change takes the new one",
          agent_model("claude", "20260922-235959")[0], "claude-opus-5-5")

    # 4. The August claude sweep must still price at its own rate card. These are
    #    its committed totals; the figure is the one published as ~$8.61.
    aug = run_cost("claude", "cli", 54576, 299472, 1691052, "20260901-013218")
    check("august claude sweep prices at opus-5 rates",
          round(aug, 2), 8.61)
    check("the same tokens at TODAY's yaml would be wrong",
          round(run_cost("claude", "cli", 54576, 299472, 1691052), 2) != round(aug, 2),
          True)

    # 5. AGENT_MODEL_HISTORY must stay in ascending date order: the lookup returns
    #    on the FIRST entry with day < changed_on, so an out-of-order or malformed
    #    date ('2026-9-22') would silently hand an older model and an older rate
    #    card to newer rows -- with no other signal that it had.
    for alias, hist in AGENT_MODEL_HISTORY.items():
        days = [h[0] for h in hist]
        check(f"AGENT_MODEL_HISTORY[{alias!r}] dates ascending", days, sorted(days))
        for d in days:
            check(f"AGENT_MODEL_HISTORY[{alias!r}] date {d!r} parses",
                  bool(re.fullmatch(r"\d{4}-\d{2}-\d{2}( \d{2}:\d{2}:\d{2})?", d)), True)

    # 6. The other half of the same defect: an alias row must never POOL two
    #    models. Before any post-change run the historical rows stay (there is
    #    nothing newer to prefer); after one, the older epoch leaves the row,
    #    so tokens from two rate cards are never summed and then billed at one.
    old, new = {"run": "20260901-013218"}, {"run": "20260923-010000"}
    check("alias with only OLD runs keeps them",
          drop_stale_epochs({("claude", "go"): [dict(old)]})[("claude", "go")],
          [old])
    check("alias with BOTH epochs drops the old one",
          drop_stale_epochs({("claude", "go"): [dict(old), dict(new)]})[("claude", "go")],
          [new])
    check("a pinned (non-alias) model is never touched",
          drop_stale_epochs({("claude-opus-5", "go"): [dict(old), dict(new)]})[("claude-opus-5", "go")],
          [old, new])

    if fails:
        sys.exit("report.py --selftest FAILED:\n" + "\n".join(fails))
    print("report.py: agent-history pricing guard OK")
    return 0


def main():
    args = sys.argv[1:]
    if "--selftest" in args:
        return selftest()
    if "--seeds" in args:
        here = pathlib.Path(__file__).resolve().parent
        return seed_value(here / "results")
    html_out = None
    if "--html" in args:
        i = args.index("--html")
        html_out = args[i + 1]
        args = args[:i] + args[i + 2:]
    md_out = None
    if "--markdown" in args:
        i = args.index("--markdown")
        md_out = args[i + 1]
        args = args[:i] + args[i + 2:]
    json_out = None
    if "--json" in args:
        i = args.index("--json")
        json_out = args[i + 1]
        args = args[:i] + args[i + 2:]
    path = pathlib.Path(args[0] if args
                        else pathlib.Path(__file__).parent / "results.csv")
    if not path.exists():
        sys.exit(f"{path}: no results yet")
    rows = list(csv.DictReader(path.open()))
    if not rows:
        sys.exit(f"{path}: no measurements yet")
    # Every value the page interpolates is either escaped or, for the run id,
    # checked against its shape here: it lands in a data-sort attribute, and a
    # row whose run field carried markup would otherwise become script in a
    # tracked, shared page (review run 20260907-150029).
    for r in rows:
        if not RUN_ID.fullmatch(r.get("run", "")):
            sys.exit(f"{path}: run id {r.get('run')!r} is not YYYYMMDD-HHMMSS")
        # Normalise the two token shapes to ONE meaning: tokens_in is FRESH
        # input, cache_read is the cached input beside it. That is what the
        # claude harness reports (Anthropic's input_tokens excludes cache
        # reads). codex reports OpenAI's shape, where usage.input_tokens is the
        # whole prompt and cached_input_tokens the cached SUBSET of it -- so
        # until 2026-09-07 every codex-route row was charged its cached input
        # twice, once at the fresh rate inside tokens_in and again at the cache
        # rate, and its token total was inflated against claude's by the same
        # amount (review run 20260907-150029; the codex probe that proved it
        # read 15,895 input with 11,008 cached for a one-line prompt). The csv
        # keeps the raw numbers; the subtraction lives here, in one place.
        if cached_is_subset(r["model"]):
            r["tokens_in"] = str(max(int(r["tokens_in"]) - int(r["cache_read"]), 0))

    # Before anything is aggregated or published: if a route started caching
    # earlier than ROUTE_CACHING claims, say so here rather than emit dollar
    # columns that silently mix two pricing eras.
    check_caching_eras(rows)

    # Rows are grouped per (model, target) across repeats, and a target's
    # representative run is chosen CLEAN-FIRST.
    #
    # A re-run after a contract failure is not a variance repeat, it is a
    # correction: the failed run lost a whole lens, so its recall is deflated
    # rather than low, and averaging it with the clean run would carry that
    # damage into the number. The failed row is still counted in `errors` and
    # in the cost columns -- the failure is the disqualifier and must not
    # disappear just because the target was measured again. Where several CLEAN
    # runs of a target exist, that IS a real repeat and they are medianed.
    by_mt = {}
    for r in rows:
        tgt = r.get("target") or ("go" if r["task"] == "code" else r["task"])
        by_mt.setdefault((r["model"], tgt), []).append(r)

    # AN AGENT ROW MUST NOT POOL TWO MODELS. Everything below sums tokens and
    # medians recall across every run in a group, and a group is keyed on the
    # csv's `model` column -- which for `claude` and `codex` is a mutable ALIAS.
    # So the first re-measurement after a seat change would put the old and the
    # new model in one bucket, median their recalls together, add their tokens,
    # and (because _asof reads the LATEST date) bill the whole mixture at the new
    # model's rates. That is the same silent restatement AGENT_MODEL_HISTORY
    # exists to stop, arriving by a different door.
    #
    # So an alias keeps only its CURRENT epoch once it has one. Before the first
    # post-change run there is nothing to drop and the row stays historical,
    # priced at its own model by _asof. Afterwards the older runs leave the row:
    # they are not lost -- they stay in results.csv, in git, and in the pinned
    # `claude-opus-5` row that exists precisely so a seat's history survives its
    # alias moving on.
    drop_stale_epochs(by_mt)

    all_targets = sorted({t for (_, t) in by_mt},
                         key=lambda t: (LEGACY_SCALE.index(t) if t in LEGACY_SCALE
                                        else len(LEGACY_SCALE), t))

    runs = {}
    for (model, tgt), rs in by_mt.items():
        clean = [r for r in rs if int(r["errors"]) == 0]
        use = clean or rs
        rep = dict(use[-1])
        rep["matched_seeds"] = str(int(statistics.median(
            int(r["matched_seeds"]) for r in use)))
        rep["_deflated"] = "" if clean else "1"
        rep["_repeats"] = str(len(use))
        # cost and failures are the whole model's, not just the chosen run's
        for col in ("tokens_in", "tokens_out", "cache_read", "duration_s",
                    "errors", "sessions"):
            rep[col] = str(sum(int(r[col]) for r in rs))
        rep["_dates"] = sorted({r["run"].split("-")[0] for r in rs})
        # Full run ids alongside the days: _asof needs the TIME, because a
        # seat can change on a day that also has runs. `dates` stays
        # day-granular because the era and the "Measured" column want days.
        rep["_runs"] = sorted({r["run"] for r in rs})
        runs.setdefault((model, "1"), {})[tgt] = rep

    per_model = {}
    for (model, repeat), tasks in sorted(runs.items()):
        scored = {t: r for t, r in tasks.items() if t in SCORED_TARGETS}
        points = sum(int(r["matched_seeds"]) for r in scored.values())
        possible = sum(int(r["seeds"]) for r in scored.values())
        per_model.setdefault(model, []).append({
            "repeat": repeat,
            "points": points,
            "possible": possible,
            # Cost columns cover every target measured, not just the scored
            # pair: the quota a sweep actually drew is the whole of it.
            "tokens": sum(int(t["tokens_in"]) + int(t["tokens_out"]) for t in tasks.values()),
            "tok_in": sum(int(t["tokens_in"]) for t in tasks.values()),
            "tok_out": sum(int(t["tokens_out"]) for t in tasks.values()),
            "cache_read": sum(int(t["cache_read"]) for t in tasks.values()),
            "errors": sum(int(t["errors"]) for t in tasks.values()),
            "seconds": sum(int(t["duration_s"]) for t in tasks.values()),
            "sessions": sum(int(t["sessions"]) for t in tasks.values()),
            "dates": sorted({d for t in tasks.values() for d in t["_dates"]}),
            "runs": sorted({r for t in tasks.values() for r in t["_runs"]}),
            # Which pricing era this candidate's dollar columns belong to, and
            # how much of its input actually came from cache. The era is what
            # the route did when it ran; the share is what its own rows record.
            "era": caching_era(route(model),
                               sorted({d for t in tasks.values() for d in t["_dates"]})),
            "cached_share": cached_share(
                sum(int(t["tokens_in"]) for t in tasks.values()),
                sum(int(t["cache_read"]) for t in tasks.values())),
            "missing": sorted(set(SCORED_TARGETS) - set(scored)),
            # recall per target, for the by-target matrix
            # errors per target too: a run that lost a LENS to a contract
            # failure has a deflated recall, not a low one. On the python target
            # the failing lens normally supplies two thirds of all credited
            # findings, so an unmarked cell reads as "weak at python" when it
            # means "measured with a third of the review discarded".
            "by_target": {t: (int(r["matched_seeds"]), int(r["seeds"]),
                              1 if r["_deflated"] else 0)
                          for t, r in tasks.items()},
        })

    ranked = sorted(per_model.items(),
                    key=lambda kv: -statistics.median(r["points"] for r in kv[1]))

    # --html and --markdown are independent outputs, not alternatives, so one
    # invocation can write both: `make bench-readme` does exactly that, and
    # until 2026-09-07 the early return here made it silently skip the README
    # whenever both flags were given -- the html was fresh, the table stale,
    # which is the drift the target exists to prevent.
    if md_out:
        # Fail before EITHER file is written: a README without the markers
        # used to abort after report.html was already regenerated, leaving the
        # exact fresh-page-stale-table drift the one-pass target exists to
        # prevent (review run 20260907-150029).
        check_markers(md_out)
    if html_out:
        write_html(ranked, all_targets, html_out)
    if md_out:
        write_markdown(ranked, all_targets, md_out)
    if json_out:
        write_json(ranked, all_targets, json_out)
    if html_out or md_out or json_out:
        return

    print(f"{'candidate':30s} {'model measured':22s} {'route':11s} {'pays':13s} "
          f"{'score':>9s} {'tok':>9s} {'pts/Mtok':>9s} {'total $':>9s} "
          f"{'per point':>10s} {'cached*':>7s} {'/pt unca':>9s} "
          f"{'sec':>6s} {'err':>4s} {'measured on':>17s}")
    print("-" * 196)
    for model, entries in ranked:
        median = statistics.median(e["points"] for e in entries)
        for e in entries:
            mtok = e["tokens"] / 1e6
            eff = e["points"] / mtok if mtok else 0.0
            flag = ""
            if e["missing"]:
                flag = f"  INCOMPLETE (no {', '.join(e['missing'])} run)"
            label = model if len(entries) == 1 else f"{model} #{e['repeat']}"
            measured = resolve(model, _asof(e))
            harness = route(model)
            print(f"{label:30s} {(measured if measured != model else '-'):22s} "
                  f"{harness:11s} {billing(model, harness):13s} "
                  f"{e['points']:>4d}/{e['possible']:<4d} {e['tokens']:>9d} {eff:>9.1f} "
                  f"{money(harness, run_cost(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], _asof(e))):>9s} "
                  f"{cost_per_point(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], e['points'], _asof(e)):>10s} "
                  f"{cached_cell(e):>7s} "
                  f"{per_point_uncached(model, harness, e):>9s} "
                  f"{e['seconds']:>6d} {e['errors']:>4d} "
                  f"{measured_on(e['dates']):>17s}{flag}")
        if len(entries) > 1:
            print(f"{'  median':30s} {'':22s} {'':11s} {'':13s} {median:>4.0f}/100")
    # --- by-target matrix: recall per target, per model ---
    #
    # The headline score collapses every target into one number, which is
    # exactly what hides a target the whole field is bad at. A column per
    # target makes "the field gets 87% on Go and 30% on Rust" visible at a
    # glance, and that gap is a fact about the TARGET -- either the language is
    # genuinely harder to review or its seeds are miscalibrated -- not about any
    # one model.
    if len(all_targets) > 1:
        print()
        # "55/60 92%" is 9 characters; +3 keeps the columns apart.
        width = max(12, max(len(t) for t in all_targets) + 3)
        head = "".join(f"{t:>{width}s}" for t in all_targets)
        print(f"{'by target (recall)':30s}{head}")
        print("-" * (30 + width * len(all_targets)))
        for model, entries in ranked:
            e = entries[0]
            cells = ""
            for t in all_targets:
                got = e["by_target"].get(t)
                if got is None:
                    cells += f"{'-':>{width}s}"
                else:
                    mark = "!" if len(got) > 2 and got[2] else ""
                    cells += f"{f'{got[0]}/{got[1]} {got[0] / got[1] * 100:.0f}%{mark}':>{width}s}"
            print(f"{model[:29]:30s}{cells}")
        print("-" * (30 + width * len(all_targets)))
        # The field's own rate per target: the number that says a target is
        # saturated (everyone near 100%) or broken/too hard (everyone near 0).
        foot = ""
        for t in all_targets:
            # A deflated cell is excluded from the median: it would drag the
            # field's rate for that target down for a reason that is about one
            # model's output contract, not about the target.
            rates = [e[0] / e[1] for m, es in ranked
                     for e in [es[0]["by_target"].get(t)]
                     if e and not (len(e) > 2 and e[2])]
            foot += f"{'-':>{width}s}" if not rates else \
                f"{f'med {statistics.median(rates) * 100:.0f}%':>{width}s}"
        print(f"{'field median':30s}{foot}")

    print()
    scales = sorted({e["possible"] for _, es in ranked for e in es})
    print(f"{len(ranked)} model(s); score is every target, out of "
          + (str(scales[0]) if len(scales) == 1 else
             f"{scales[0]}-{scales[-1]} (SCALES DIFFER -- rows are not comparable)"))
    print(f"scored targets: {', '.join(SCORED_TARGETS)}. A row short of any of "
          "them is marked INCOMPLETE and its")
    print("total is not comparable with a full row. The go+design pair the "
          "older /90 headline used is still")
    print("readable as the first two columns of the matrix above.")
    print("per point: cost of one seeded defect found. $ = money actually "
          "billed to a card (OpenRouter only).")
    print("           ~$ = notional: what the run would cost at published "
          "rates, on a route that bills a plan")
    print("           rather than the tokens -- claude, codex AND ollama, "
          "which still meters a session and a")
    print("           weekly quota on top of its per-token rates. k = tokens, "
          "for a model with no published rate.")


if __name__ == "__main__":
    main()
