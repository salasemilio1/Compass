"""Score the Q2 arms once (protocol: experiments/2026-10-22-retrieval-q2-keywords.md).

nDCG@10 on the graded pool (unjudged = 0), MRR and recall@5 of the first grade-2 record; paired
differences with a bootstrap 95% interval over tasks (10,000 resamples, seed 1). Also the share
of query words that appear in the task's keywords but not in its record (A and D sharing a source).
"""
import glob
import json
import math
import random
import re
import statistics

key = json.load(open("C-key.json"))
grades = {}
for f in sorted(glob.glob("C/grades-*.jsonl")):
    for line in open(f):
        if line.strip():
            row = json.loads(line)
            grades[(row["id"].split("-R")[0], key[row["id"]])] = int(row["grade"])
missing = [rid for rid in key if (rid.split("-R")[0], key[rid]) not in grades]
runs = json.load(open("runs.json"))
queries = [json.loads(l) for l in open("queries.jsonl")]


def metrics(arm):
    per_task = {}
    for task in queries:
        tid = task["id"]
        ideal = sorted((g for (t, _), g in grades.items() if t == tid), reverse=True)[:10]
        idcg = sum((2 ** g - 1) / math.log2(i + 2) for i, g in enumerate(ideal))
        rows = []
        for q in task["queries"]:
            ranking = runs[arm][q]
            gs = [grades.get((tid, p), 0) for p in ranking[:10]]
            dcg = sum((2 ** g - 1) / math.log2(i + 2) for i, g in enumerate(gs))
            first = next((i + 1 for i, g in enumerate(gs) if g == 2), None)
            rows.append((dcg / idcg if idcg else 0.0, 1 / first if first else 0.0, 1.0 if first and first <= 5 else 0.0))
        per_task[tid] = [statistics.mean(c) for c in zip(*rows)]
    return per_task


def interval(a, b, index, seed=1, n=10000):
    tasks = sorted(a)
    diffs = [a[t][index] - b[t][index] for t in tasks]
    rng = random.Random(seed)
    boots = sorted(statistics.mean(rng.choice(diffs) for _ in diffs) for _ in range(n))
    return statistics.mean(diffs), boots[int(0.025 * n)], boots[int(0.975 * n)]


if __name__ == "__main__":
    print("graded", len(grades), "missing", len(missing))
    tasks_without_2 = sum(1 for task in queries if not any(g == 2 for (t, _), g in grades.items() if t == task["id"]))
    print("tasks with no grade-2 record in the pool:", tasks_without_2)
    results = {arm: metrics(arm) for arm in runs}
    print(f"{'arm':12} nDCG@10  MRR(2)  r@5(2)")
    for arm, per_task in results.items():
        means = [statistics.mean(v[i] for v in per_task.values()) for i in range(3)]
        print(f"{arm:12} {means[0]:.3f}    {means[1]:.3f}   {means[2]:.3f}")
    for a, b in (("hygiene", "fts5"), ("hygiene", "bm25f"), ("hygiene+kw", "hygiene"), ("fts5+kw", "fts5"),
                 ("rg+kw", "rg"), ("blend", "hygiene"), ("blend", "fts5"), ("fts5", "bm25f")):
        d, lo, hi = interval(results[a], results[b], 0)
        print(f"nDCG@10 {a} - {b}: {d:+.3f} [{lo:+.3f}, {hi:+.3f}]")
