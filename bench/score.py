#!/usr/bin/env python3
"""Score one fixpoint bench run against a seed manifest.

Usage: bench/score.py <manifest.yaml> <summary-*.json> <model> [repeat]

Reads the run summary fixpoint already writes, matches every finding against
the manifest's seeded defects, and appends one row to bench/results.csv plus a
per-seed hit table to bench/results/. Matching is deterministic: right file,
within +/- span lines of the seed (skipped when line is 0), and at least
`need` of the keywords present in title+description, case-insensitive.

No YAML dependency: the manifest is parsed with a purpose-built reader that
understands exactly the shape bench/manifest-*.yaml uses.
"""

import csv
import json
import pathlib
import re
import sys


def read_manifest(path):
    """Parse the two-level manifest without a YAML library."""
    task, seeds, cur = None, [], None
    for raw in pathlib.Path(path).read_text().splitlines():
        line = raw.split(" #")[0].rstrip() if not raw.lstrip().startswith("#") else ""
        if not line.strip():
            continue
        if line.startswith("task:"):
            task = line.split(":", 1)[1].strip()
        elif line.strip().startswith("- id:"):
            cur = {"id": line.split(":", 1)[1].strip()}
            seeds.append(cur)
        elif cur is not None and ":" in line:
            key, val = (s.strip() for s in line.strip().split(":", 1))
            if key == "keywords":
                words = re.findall(r'"([^"]+)"|([^,\[\]]+)', val.strip("[]"))
                cur[key] = [a or b.strip() for a, b in words if (a or b.strip())]
            elif key in ("line", "span", "need"):
                cur[key] = int(val)
            else:
                cur[key] = val.strip('"')
    if task is None or not seeds:
        sys.exit(f"{path}: no task or no seeds parsed")
    return task, seeds


def matches(seed, finding):
    # `file` may list alternatives ("store.go|worker.go") for a defect whose
    # two halves live in different files and get reported from either.
    if not any(finding.get("file", "").endswith(f) for f in seed["file"].split("|")):
        return False
    if seed["line"] > 0 and abs(int(finding.get("line") or 0) - seed["line"]) > seed["span"]:
        return False
    text = (finding.get("title", "") + " " + finding.get("description", "")).lower()
    # Word-boundary prefix match: "race" hits "races" and "racing" but "lock"
    # does not hit "block" -- substring matching produced both failure modes
    # in calibration.
    hits = sum(1 for k in seed["keywords"]
               if re.search(r"\b" + re.escape(k.lower()), text))
    return hits >= seed["need"]


def main():
    if len(sys.argv) < 4:
        sys.exit(__doc__)
    manifest_path, summary_path, model = sys.argv[1], sys.argv[2], sys.argv[3]
    repeat = sys.argv[4] if len(sys.argv) > 4 else "1"

    task, seeds = read_manifest(manifest_path)
    summary = json.loads(pathlib.Path(summary_path).read_text())
    rounds = summary.get("rounds") or []
    findings = [f for r in rounds for f in (r.get("findings") or [])]
    steps = [s for r in rounds for s in (r.get("steps") or [])]

    hit = {s["id"]: [] for s in seeds}
    extras = []
    for f in findings:
        matched = False
        for s in seeds:
            if matches(s, f):
                hit[s["id"]].append(f)
                matched = True
        if not matched:
            extras.append(f)

    found = sum(1 for v in hit.values() if v)
    recall = found / len(seeds)
    tok_in = sum(s.get("usage", {}).get("input_tokens", 0) for s in steps)
    tok_out = sum(s.get("usage", {}).get("output_tokens", 0) for s in steps)
    tok_cache = sum(s.get("usage", {}).get("cache_read_tokens", 0) for s in steps)
    duration_s = sum(s.get("duration_ms", 0) for s in steps) / 1000
    sessions = len(steps)
    # A session that produced no finding AND no parseable output is an error;
    # fixpoint's summary carries errors per reviewer only in the table, so the
    # observable here is: sessions whose output_bytes are 0.
    errors = sum(1 for s in steps if s.get("output_bytes", 0) == 0)
    mtok = (tok_in + tok_out) / 1e6
    eff = round(found / mtok, 2) if mtok else 0.0

    run_id = pathlib.Path(summary_path).parent.name
    outdir = pathlib.Path(__file__).parent / "results"
    outdir.mkdir(exist_ok=True)

    csv_path = pathlib.Path(__file__).parent / "results.csv"
    new = not csv_path.exists()
    with csv_path.open("a", newline="") as fh:
        w = csv.writer(fh)
        if new:
            w.writerow(["run", "task", "model", "repeat", "sessions", "errors",
                        "findings", "matched_seeds", "seeds", "recall", "extras",
                        "tokens_in", "tokens_out", "cache_read", "duration_s", "found_per_mtok"])
        w.writerow([run_id, task, model, repeat, sessions, errors,
                    len(findings), found, len(seeds), f"{recall:.2f}", len(extras),
                    tok_in, tok_out, tok_cache, int(duration_s), eff])

    lines = [f"# {model} · {task} · run {run_id} (repeat {repeat})", ""]
    lines.append(f"recall **{found}/{len(seeds)}** · {len(findings)} finding(s), "
                 f"{len(extras)} unmatched · {tok_in + tok_out} tokens · {int(duration_s)}s")
    lines.append("")
    lines.append("| seed | found | note | matched by |")
    lines.append("|---|---|---|---|")
    for s in seeds:
        got = hit[s["id"]]
        by = "; ".join(f"{g.get('lens', '?')}: {g.get('title', '')[:60]}" for g in got[:2])
        lines.append(f"| {s['id']} | {'YES' if got else '—'} | {s.get('note', '')[:70]} | {by} |")
    if extras:
        lines.append("")
        lines.append("Unmatched findings (noise, or genuinely new — skim before dismissing):")
        for f in extras:
            lines.append(f"- ({f.get('severity', '?')}) {f.get('file', '?')}:{f.get('line', '?')} "
                         f"— {f.get('title', '')[:90]}")
    report = outdir / f"{model.replace(':', '_').replace('/', '_')}-{task}-{run_id}.md"
    report.write_text("\n".join(lines) + "\n")

    print(f"bench: {model} {task}: recall {found}/{len(seeds)}, {len(extras)} unmatched, "
          f"{tok_in + tok_out} tok, {int(duration_s)}s -> {report}")


if __name__ == "__main__":
    main()
