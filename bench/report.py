#!/usr/bin/env python3
"""Combine bench/results.csv into one score per model, out of the live seed pool.

Usage: bench/report.py [results.csv]
       bench/report.py [results.csv] --html <out.html>

The benchmark's points are split across two calibrated TARGETS -- the Go target
and the design target -- so a model's score is
only complete once both have run. Further language targets are reported in the
by-target matrix rather than folded into that total: a score is comparable only
to runs sharing its scale. Repeats are shown separately and then as a
median, because single-run recall on a hundred seeds is noisy and the variance
is itself information: a model that scores 61 then 38 is worse than one that
scores 49 twice.

A row missing one of the two targets is reported as incomplete rather than
silently scored out of one target's pool. The denominator comes from each row's
own `seeds` column, so a retired seed changes the scale everywhere at once.
"""

import csv
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

# The calibrated pair the published /100 has always meant. New language targets
# are reported per-target instead of being folded into this total, because a
# score is only comparable to the runs it shares a scale with -- and every seat
# decision on record quotes this one.
LEGACY_SCALE = ("go", "design")
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
        return "credits"
    if harness == "ollama":
        base = model.split(":", 1)[0]
        return "sub+credits" if base in OLLAMA_CREDIT_MODELS else "sub (ollama)"
    if model == "codex":
        return "sub (chatgpt)"
    return "sub (claude)"


def cost_per_point(model, harness, tok_in, tok_out, cache_read, points):
    """What one seeded defect cost, in the currency that route actually spends.

    A dollar figure wherever a published rate exists, prefixed with ~ when the
    route is a subscription and the money is therefore notional (nothing is
    billed per token there); bare where prepaid credits are actually spent. Ollama has no
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
        failures += 1 if e["errors"] else 0
        mtok = e["tokens"] / 1e6
        eff = e["points"] / mtok if mtok else 0.0
        harness = route(model)
        measured = resolve(model)
        baseline = model in ("claude", "codex")
        width = 100.0 * e["points"] / max(top, 1)
        incomplete = "  (code only)" if e["missing"] else ""
        rows_html.append(f"""      <tr class="{'failed' if e['errors'] else ''}">
        <td class="rank">{i}</td>
        <td class="left">
          <div class="name">{_esc(model)}{'<span class="tag">baseline</span>' if baseline else ''}</div>
          <div class="sub">{_esc(measured) if measured != model else ''}</div>
        </td>
        <td class="left sub">{_esc(harness)}</td>
        <td class="left sub">{_esc(billing(model, harness))}</td>
        <td class="score">
          <div class="bar" style="width: {width:.1f}%"></div>
          <span>{e['points']}<span class="of">/{e['possible']}{incomplete}</span></span>
        </td>
        <td>{e['tokens']:,}</td>
        <td>{eff:.0f}</td>
        <td>{_esc(cost_per_point(model, harness, e['tok_in'], e['tok_out'], e['cache_read'], e['points']))}</td>
        <td>{e['seconds']:,}</td>
        <td class="{'err' if e['errors'] else 'zero'}">{e['errors']}</td>
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
                    cells += '<td class="zero">&mdash;</td>'
                else:
                    pct = got[0] / got[1] * 100
                    cells += (f'<td><span class="pct">{pct:.0f}%</span>'
                              f'<span class="sub"> {got[0]}/{got[1]}</span></td>')
            body.append(f'      <tr><td class="left"><div class="name">'
                        f'{_esc(model)}</div></td>{cells}</tr>')
        foot = ""
        for t in all_targets:
            rates = [e[0] / e[1] for m, es in ranked
                     for e in [es[0]["by_target"].get(t)] if e]
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
    <p class="lede">Every candidate reviews the same two frozen targets and is scored against
    <strong>100 planted defects</strong> &mdash; 60 in a Go service, 40 in a design document.
    No judge, no taste: a deterministic matcher credits each finding to at most one seed.
    Nobody scores 100, and that is the point. What matters is the distance to the
    <strong>claude baseline</strong>, and what the points cost.</p>
  </header>

  <dl class="facts">
    <div class="fact"><dt>Candidates</dt><dd>{len(ranked)}</dd></div>
    <div class="fact"><dt>Sessions</dt><dd>{sessions}</dd></div>
    <div class="fact"><dt>Points available</dt><dd>100</dd></div>
    <div class="fact"><dt>Repeats</dt><dd>{repeats}</dd></div>
    <div class="fact"><dt>Contract failures</dt><dd>{failures}</dd></div>
  </dl>

  <div class="scroller">
    <table>
      <caption>Ranked by score. One repeat each &mdash; single-run recall is a sample, not a verdict.</caption>
      <thead>
        <tr>
          <th class="left" scope="col">#</th>
          <th class="left" scope="col">Candidate</th>
          <th class="left" scope="col">Route</th>
          <th class="left" scope="col">Pays</th>
          <th class="left" scope="col">Score</th>
          <th scope="col">Tokens</th>
          <th scope="col">Pts / Mtok</th>
          <th scope="col">Per point</th>
          <th scope="col">Wall (s)</th>
          <th scope="col">Err</th>
        </tr>
      </thead>
      <tbody>
{chr(10).join(rows_html)}
      </tbody>
    </table>
  </div>

  {matrix_html}

  <div class="notes">
    <p><b>Per point</b> is what one found defect cost. <code>$</code> is spent from prepaid
    credits and appears on an invoice; <code>~$</code> is notional &mdash; the subscription
    routes bill nothing per token, so the figure is what the same run would have cost at API
    rates, which is the only way to compare a seat paid in subscription against one paid in
    credits. <code>k</code> is thousands of tokens: ollama sells quota, not tokens, so a dollar
    figure there would be invented.</p>
    <p><b>Contract failure</b> is the disqualifier, independent of score: a session that
    returns unparseable output burns a full slot and can flip a panel verdict to inconclusive.
    Both models that failed here have failed before.</p>
    <p><b>Wall clock</b> matters as much as tokens: a panel round runs at the pace of its
    slowest reviewer.</p>
  </div>
</div>
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


def main():
    args = sys.argv[1:]
    if "--seeds" in args:
        here = pathlib.Path(__file__).resolve().parent
        return seed_value(here / "results")
    html_out = None
    if "--html" in args:
        i = args.index("--html")
        html_out = args[i + 1]
        args = args[:i] + args[i + 2:]
    path = pathlib.Path(args[0] if args
                        else pathlib.Path(__file__).parent / "results.csv")
    if not path.exists():
        sys.exit(f"{path}: no results yet")
    rows = list(csv.DictReader(path.open()))
    if not rows:
        sys.exit(f"{path}: no measurements yet")

    runs = {}
    for r in rows:
        key = (r["model"], r["repeat"])
        # Rows written before the target column carry only a task; a code row
        # then meant the Go target, the only one that existed.
        tgt = r.get("target") or ("go" if r["task"] == "code" else r["task"])
        runs.setdefault(key, {})[tgt] = r

    all_targets = sorted({t for tasks in runs.values() for t in tasks},
                         key=lambda t: (LEGACY_SCALE.index(t) if t in LEGACY_SCALE
                                        else len(LEGACY_SCALE), t))

    per_model = {}
    for (model, repeat), tasks in sorted(runs.items()):
        scored = {t: r for t, r in tasks.items() if t in LEGACY_SCALE}
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
            "dates": sorted({t["run"].split("-")[0] for t in tasks.values()}),
            "missing": sorted(set(LEGACY_SCALE) - set(scored)),
            # recall per target, for the by-target matrix
            "by_target": {t: (int(r["matched_seeds"]), int(r["seeds"]))
                          for t, r in tasks.items()},
        })

    ranked = sorted(per_model.items(),
                    key=lambda kv: -statistics.median(r["points"] for r in kv[1]))

    if html_out:
        write_html(ranked, all_targets, html_out)
        return

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
                cells += f"{'-':>{width}s}" if got is None else \
                    f"{f'{got[0]}/{got[1]} {got[0] / got[1] * 100:.0f}%':>{width}s}"
            print(f"{model[:29]:30s}{cells}")
        print("-" * (30 + width * len(all_targets)))
        # The field's own rate per target: the number that says a target is
        # saturated (everyone near 100%) or broken/too hard (everyone near 0).
        foot = ""
        for t in all_targets:
            rates = [e[0] / e[1] for m, es in ranked
                     for e in [es[0]["by_target"].get(t)] if e]
            foot += f"{'-':>{width}s}" if not rates else \
                f"{f'med {statistics.median(rates) * 100:.0f}%':>{width}s}"
        print(f"{'field median':30s}{foot}")

    print()
    scales = sorted({e["possible"] for _, es in ranked for e in es})
    print(f"{len(ranked)} model(s); score is {'+'.join(LEGACY_SCALE)}, out of "
          + (str(scales[0]) if len(scales) == 1 else
             f"{scales[0]}-{scales[-1]} (SCALES DIFFER -- rows are not comparable)"))
    if len(all_targets) > len(LEGACY_SCALE):
        extra = [t for t in all_targets if t not in LEGACY_SCALE]
        print(f"score covers {'+'.join(LEGACY_SCALE)} only, the calibrated scale; "
              f"{', '.join(extra)} are reported per target above")
    print("per point: cost of one seeded defect found. $ = spent from prepaid "
          "credits, ~$ = notional (subscription,")
    print("           nothing billed per token), k = thousands of tokens "
          "(ollama sells quota, not tokens)")


if __name__ == "__main__":
    main()
