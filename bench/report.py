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
        tokens = sum(int(t["tokens_in"]) + int(t["tokens_out"]) for t in tasks.values())
        errors = sum(int(t["errors"]) for t in tasks.values())
        seconds = sum(int(t["duration_s"]) for t in tasks.values())
        per_model.setdefault(model, []).append({
            "repeat": repeat, "points": points, "possible": possible,
            "tokens": tokens, "errors": errors, "seconds": seconds,
            "missing": sorted(set(TASK_POINTS) - set(tasks)),
        })

    ranked = sorted(per_model.items(),
                    key=lambda kv: -statistics.median(r["points"] for r in kv[1]))

    print(f"{'model':42s} {'score':>9s} {'tok':>9s} {'pts/Mtok':>9s} {'sec':>6s} {'err':>4s}")
    print("-" * 84)
    for model, entries in ranked:
        median = statistics.median(e["points"] for e in entries)
        for e in entries:
            mtok = e["tokens"] / 1e6
            eff = e["points"] / mtok if mtok else 0.0
            flag = ""
            if e["missing"]:
                flag = f"  INCOMPLETE (no {', '.join(e['missing'])} run)"
            label = model if len(entries) == 1 else f"{model} #{e['repeat']}"
            print(f"{label:42s} {e['points']:>4d}/{e['possible']:<4d} {e['tokens']:>9d} "
                  f"{eff:>9.1f} {e['seconds']:>6d} {e['errors']:>4d}{flag}")
        if len(entries) > 1:
            print(f"{'  median':42s} {median:>4.0f}/100")
    print()
    print(f"{len(ranked)} model(s); 100 points = {TASK_POINTS['code']} code seeds "
          f"+ {TASK_POINTS['design']} design seeds")


if __name__ == "__main__":
    main()
