#!/usr/bin/env python3
"""Score one fixpoint bench run against a seed manifest.

Usage: bench/score.py <manifest.yaml> <summary-*.json> <model> [repeat]
       bench/score.py --check <manifest.yaml>
       bench/score.py --selftest <manifest.yaml>

Reads the run summary fixpoint already writes, matches every finding against
the manifest's seeded defects, and appends one row to bench/results.csv plus a
per-seed hit table to bench/results/. Matching is deterministic: right file,
within +/- span lines of the seed (skipped when the seed has no line), and at
least `need` of the keywords present in title+description, case-insensitive.

Every finding is credited to AT MOST ONE seed -- the one it matches best (most
keywords, then closest line). With 60 code seeds packed into a few files, the
older any-seed-that-matches rule let one vague finding light up three
neighbouring seeds and inflate recall; a reviewer is credited for what it
actually distinguished.

Seed lines come from `anchor`, an exact and unique substring of the seeded
line, resolved against the target under the manifest's `root` at scoring time.
Hand-counted line numbers rot silently the moment the target is touched; an
anchor that no longer resolves is a hard error. `--check` runs that resolution
alone, which is how you validate a manifest edit without paying for a run.

No YAML dependency: the manifest is parsed with a purpose-built reader that
understands exactly the shape bench/manifest-*.yaml uses.
"""

import csv
import json
import pathlib
import re
import sys


def parse_list(val):
    """Split a YAML flow list into its items, honouring quotes.

    Hand-rolled because the manifest is read without a YAML library. The regex
    this replaces kept the quotation marks on every quoted item that was not
    the first in the list ('"rows 1..N-1"'), and a keyword carrying its own
    quotes can never match a finding -- those keywords were silently dead, which
    biased every measurement made before 2026-08-20 towards false negatives.
    """
    items, cur, quote = [], "", None
    for ch in val.strip().strip("[]"):
        if quote:
            if ch == quote:
                items.append(cur)
                cur, quote = "", None
            else:
                cur += ch
        elif ch in "\"'":
            # Drop the separator whitespace sitting in front of a quoted item,
            # so the keyword is exactly what the quotes contain.
            if not cur.strip():
                cur = ""
            quote = ch
        elif ch == ",":
            if cur.strip():
                items.append(cur.strip())
            cur = ""
        else:
            cur += ch
    if cur.strip():
        items.append(cur.strip())
    return [i for i in items if i]


def read_manifest(path):
    """Parse the two-level manifest without a YAML library."""
    task, target, root, seeds, cur = None, None, None, [], None
    for raw in pathlib.Path(path).read_text().splitlines():
        line = raw.split(" #")[0].rstrip() if not raw.lstrip().startswith("#") else ""
        if not line.strip():
            continue
        if line.startswith("task:"):
            task = line.split(":", 1)[1].strip()
        elif line.startswith("target:"):
            target = line.split(":", 1)[1].strip()
        elif line.startswith("root:"):
            root = line.split(":", 1)[1].strip()
        elif line.strip().startswith("- id:"):
            cur = {"id": line.split(":", 1)[1].strip()}
            seeds.append(cur)
        elif cur is not None and ":" in line:
            key, val = (s.strip() for s in line.strip().split(":", 1))
            if key == "keywords":
                cur[key] = parse_list(val)
            elif key in ("line", "span", "need"):
                cur[key] = int(val)
            elif key == "retired":
                cur[key] = val.strip().lower() in ("true", "yes", "1")
            else:
                # Anchors carry code, so they are quoted with whichever quote
                # the code itself does not use.
                val = val.strip()
                if len(val) >= 2 and val[0] == val[-1] and val[0] in "\"'":
                    val = val[1:-1]
                cur[key] = val
    if task is None or not seeds:
        sys.exit(f"{path}: no task or no seeds parsed")
    # A manifest that predates the target dimension scores under its task name,
    # which is what the design target's own target value already is.
    if target is None:
        target = task
    ids = [s["id"] for s in seeds]
    if len(set(ids)) != len(ids):
        dupes = sorted({i for i in ids if ids.count(i) > 1})
        sys.exit(f"{path}: duplicate seed ids: {', '.join(dupes)}")
    for seed in seeds:
        seed.setdefault("line", 0)
        seed.setdefault("span", 0)
        seed.setdefault("need", 1)
        if not seed.get("keywords"):
            sys.exit(f"{path}: seed {seed['id']} has no keywords")
    return task, target, root, seeds


def resolve_anchors(manifest_path, root, seeds):
    """Turn every seed's `anchor` into a line number in the target.

    A seed whose anchor is missing, ambiguous, or in the wrong file is a
    manifest that no longer describes the target: that is a hard error, not a
    seed nobody finds. Seeds without an anchor keep whatever `line` they carry
    (0 means no line gate at all).
    """
    if root is None:
        return []
    base = pathlib.Path(manifest_path).parent / root
    problems, cache = [], {}
    for seed in seeds:
        anchor = seed.get("anchor")
        if not anchor:
            continue
        name = seed["file"]
        if "|" in name:
            problems.append(f"{seed['id']}: anchor with a multi-file seed ({name})")
            continue
        path = base / name
        if name not in cache:
            if not path.exists():
                problems.append(f"{seed['id']}: {path} does not exist")
                cache[name] = []
            else:
                cache[name] = path.read_text().splitlines()
        lines = cache[name]
        found = [i + 1 for i, text in enumerate(lines) if anchor in text]
        if len(found) == 0:
            problems.append(f"{seed['id']}: anchor not found in {name}: {anchor!r}")
        elif len(found) > 1:
            problems.append(
                f"{seed['id']}: anchor is ambiguous in {name} (lines {found}): {anchor!r}")
        else:
            seed["line"] = found[0]
    return problems


def assign(seeds, findings):
    """Credit every finding to at most one seed: its best match.

    Ties go to the seed listed first in the manifest, which is why two seeds
    that a finding can satisfy equally well must be told apart by keywords --
    `--selftest` is what catches the pairs that cannot.
    """
    hit = {s["id"]: [] for s in seeds}
    extras = []
    for f in findings:
        best, best_key = None, None
        for s in seeds:
            key = match_score(s, f)
            if key is not None and (best_key is None or key > best_key):
                best, best_key = s, key
        if best is None:
            extras.append(f)
        else:
            hit[best["id"]].append(f)
    return hit, extras


def match_score(seed, finding):
    """Return the ranking key if the finding matches this seed, else None.

    The key decides which seed a finding belongs to when several would accept
    it: most keywords first, then the seed that gates on a line (evidence about
    one place beats a file-wide seed), then the closer line.
    """
    # `file` may list alternatives ("store.go|worker.go") for a defect whose
    # two halves live in different files and get reported from either.
    if not any(finding.get("file", "").endswith(f) for f in seed["file"].split("|")):
        return None
    distance = 0
    if seed["line"] > 0:
        distance = abs(int(finding.get("line") or 0) - seed["line"])
        if distance > seed["span"]:
            return None
    text = (finding.get("title", "") + " " + finding.get("description", "")).lower()
    # Word-boundary prefix match: "race" hits "races" and "racing" but "lock"
    # does not hit "block" -- substring matching produced both failure modes
    # in calibration.
    hits = sum(1 for k in seed["keywords"]
               if re.search(r"\b" + re.escape(k.lower()), text))
    if hits < seed["need"]:
        return None
    return (hits, 1 if seed["line"] > 0 else 0, -distance)


def check(manifest_path):
    """Validate a manifest against its target without running anything."""
    task, target, root, seeds = read_manifest(manifest_path)
    problems = resolve_anchors(manifest_path, root, seeds)
    anchored = sum(1 for s in seeds if s.get("anchor"))
    live = sum(1 for s in seeds if not s.get("retired"))
    print(f"{manifest_path}: task {task}, {len(seeds)} seeds "
          f"({live} scored, {len(seeds) - live} retired), "
          f"{anchored} anchored, {len(seeds) - anchored} matched on keywords alone")
    for p in problems:
        print(f"  BROKEN {p}")
    if problems:
        return 1
    for seed in sorted(seeds, key=lambda s: (s["file"], s["line"])):
        where = f"{seed['file']}:{seed['line']}+/-{seed['span']}" if seed["line"] else seed["file"]
        print(f"  {seed['id']:5s} {where:34s} need {seed['need']} of {len(seed['keywords'])}")
    return 0


def selftest(manifest_path):
    """Prove every seed can be found without stealing another seed's finding.

    Each seed is turned into the most on-the-nose finding it could receive --
    its own keywords and its own note, at its own line -- and the whole set is
    matched at once. A seed that does not get its own finding back is a seed
    two others cannot be told apart from, which at 100 seeds is the failure
    mode that quietly caps everyone's recall.
    """
    task, target, root, seeds = read_manifest(manifest_path)
    problems = resolve_anchors(manifest_path, root, seeds)
    if problems:
        for p in problems:
            print(f"  BROKEN {p}")
        return 1
    findings = [{
        "file": s["file"].split("|")[0],
        "line": s["line"],
        "title": f"seed {s['id']}",
        "description": " ".join(s["keywords"][:s["need"]]) + " -- " + s.get("note", ""),
    } for s in seeds]
    hit, extras = assign(seeds, findings)
    stolen = []
    for seed, f in zip(seeds, findings):
        got = hit[seed["id"]]
        if not any(g is f for g in got):
            thief = next((sid for sid, fs in hit.items() if any(g is f for g in fs)), "nothing")
            stolen.append(f"{seed['id']}: its own finding was credited to {thief}")
    print(f"{manifest_path}: selftest {len(seeds) - len(stolen)}/{len(seeds)} seeds "
          f"recover their own finding, {len(extras)} unmatched")
    for line in stolen:
        print(f"  COLLISION {line}")
    return 1 if stolen else 0


def main():
    if len(sys.argv) == 3 and sys.argv[1] == "--check":
        sys.exit(check(sys.argv[2]))
    if len(sys.argv) == 3 and sys.argv[1] == "--selftest":
        sys.exit(selftest(sys.argv[2]))
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    manifest_path, summary_path, model = sys.argv[1], sys.argv[2], sys.argv[3]
    repeat = sys.argv[4] if len(sys.argv) > 4 else "1"

    task, target, root, seeds = read_manifest(manifest_path)
    problems = resolve_anchors(manifest_path, root, seeds)
    if problems:
        # Scoring against a manifest that no longer fits the target would
        # publish a low recall as a fact about the model.
        sys.exit("bench: manifest does not match the target:\n  " + "\n  ".join(problems))
    summary = json.loads(pathlib.Path(summary_path).read_text())
    rounds = summary.get("rounds") or []
    findings = [f for r in rounds for f in (r.get("findings") or [])]
    steps = [s for r in rounds for s in (r.get("steps") or [])]

    # Retired seeds still take part in matching, deliberately. The defect is
    # still in the frozen target, so a reviewer will still report it; if the
    # seed were simply deleted, that correct finding would land in `extras` and
    # be counted as NOISE against the model. Retiring a seed must sharpen the
    # scale, not manufacture false noise -- so the seed keeps absorbing its
    # finding and only leaves the denominator.
    hit, extras = assign(seeds, findings)

    active = [s for s in seeds if not s.get("retired")]
    retired = [s for s in seeds if s.get("retired")]
    found = sum(1 for s in active if hit[s["id"]])
    recall = found / len(active) if active else 0.0
    tok_in = sum(s.get("usage", {}).get("input_tokens", 0) for s in steps)
    tok_out = sum(s.get("usage", {}).get("output_tokens", 0) for s in steps)
    tok_cache = sum(s.get("usage", {}).get("cache_read_tokens", 0) for s in steps)
    duration_s = sum(s.get("duration_ms", 0) for s in steps) / 1000
    sessions = len(steps)
    # Contract compliance is the benchmark's disqualifier, so it reads the
    # field fixpoint already sets: StepStat.failed, true whenever a session
    # errored, timed out, or broke the output contract. Counting only
    # zero-byte output (the first version of this scorer) recorded a malformed
    # NON-EMPTY reply as a success and could promote an unreliable reviewer --
    # exactly what the disqualifier exists to prevent (review run
    # 20260813-124710).
    errors = sum(1 for s in steps if s.get("failed") or s.get("output_bytes", 0) == 0)
    # A run that never reached its panel (a preflight refusal, a dead agent at
    # startup) has no steps at all. That is a contract failure too -- the
    # loudest kind -- and must not read as a clean zero.
    if not steps:
        errors = 1
    mtok = (tok_in + tok_out) / 1e6
    eff = round(found / mtok, 2) if mtok else 0.0

    run_id = pathlib.Path(summary_path).parent.name
    outdir = pathlib.Path(__file__).parent / "results"
    outdir.mkdir(exist_ok=True)

    # Keep a re-scoring extract, so a manifest change never costs a sweep.
    #
    # This exists because it already did. The 2026-09-01 retire/replace could
    # re-score only 15 of 43 runs onto the new scale: the other 28 runs'
    # .fixpoint directories had been cleaned up, so 14 models -- including the
    # seated one -- had nothing left to re-score and had to be archived on a
    # scale that no longer exists. Every seed edit before this was one `rm -rf
    # .fixpoint` away from costing a full re-sweep.
    #
    # Only the fields this scorer reads are kept, which is a fraction of the
    # summary: findings and per-step usage, not prompts, diffs or agent output.
    # A full summary averages ~75 KB and carries the reviewed code with it.
    keep = {"run": run_id, "task": task, "target": target, "model": model,
            "repeat": repeat, "rounds": []}
    for r in rounds:
        keep["rounds"].append({
            "findings": [{k: f.get(k) for k in
                          ("file", "line", "title", "description", "lens", "severity")}
                         for f in (r.get("findings") or [])],
            "steps": [{"usage": s_.get("usage", {}), "duration_ms": s_.get("duration_ms", 0),
                       "failed": s_.get("failed", False),
                       "output_bytes": s_.get("output_bytes", 0)}
                      for s_ in (r.get("steps") or [])],
        })
    keepdir = pathlib.Path(__file__).parent / "summaries"
    keepdir.mkdir(exist_ok=True)
    (keepdir / f"{run_id}-{target}.json").write_text(json.dumps(keep, indent=1))

    csv_path = pathlib.Path(__file__).parent / "results.csv"
    new = not csv_path.exists()
    with csv_path.open("a", newline="") as fh:
        # csv.writer defaults to CRLF; git normalises it away on commit, so the
        # working copy would differ from HEAD after every single run.
        w = csv.writer(fh, lineterminator="\n")
        if new:
            w.writerow(["run", "task", "target", "model", "repeat", "sessions", "errors",
                        "findings", "matched_seeds", "seeds", "recall", "extras",
                        "tokens_in", "tokens_out", "cache_read", "duration_s", "found_per_mtok"])
        w.writerow([run_id, task, target, model, repeat, sessions, errors,
                    len(findings), found, len(active), f"{recall:.2f}", len(extras),
                    tok_in, tok_out, tok_cache, int(duration_s), eff])

    lines = [f"# {model} · {target} · run {run_id} (repeat {repeat})", ""]
    lines.append(f"recall **{found}/{len(active)}** · {len(findings)} finding(s), "
                 f"{len(extras)} unmatched · {tok_in + tok_out} tokens · {int(duration_s)}s"
                 + (f" · {len(retired)} retired seed(s) not scored" if retired else ""))
    lines.append("")
    lines.append("| seed | found | note | matched by |")
    lines.append("|---|---|---|---|")
    for s in active:
        got = hit[s["id"]]
        by = "; ".join(f"{g.get('lens', '?')}: {g.get('title', '')[:60]}" for g in got[:2])
        lines.append(f"| {s['id']} | {'YES' if got else '—'} | {s.get('note', '')[:70]} | {by} |")
    if retired:
        lines.append("")
        lines.append("Retired seeds — still matched so their findings are not counted as "
                     "noise, but out of the denominator:")
        lines.append("")
        lines.append("| seed | found | note |")
        lines.append("|---|---|---|")
        for s in retired:
            got = hit[s["id"]]
            lines.append(f"| {s['id']} | {'YES' if got else '—'} | {s.get('note', '')[:70]} |")
    if extras:
        lines.append("")
        lines.append("Unmatched findings (noise, or genuinely new — skim before dismissing):")
        for f in extras:
            lines.append(f"- ({f.get('severity', '?')}) {f.get('file', '?')}:{f.get('line', '?')} "
                         f"— {f.get('title', '')[:90]}")
    report = outdir / f"{model.replace(':', '_').replace('/', '_')}-{target}-{run_id}.md"
    report.write_text("\n".join(lines) + "\n")

    print(f"bench: {model} {target}: recall {found}/{len(active)}, {len(extras)} unmatched, "
          f"{tok_in + tok_out} tok, {int(duration_s)}s -> {report}")


if __name__ == "__main__":
    main()
