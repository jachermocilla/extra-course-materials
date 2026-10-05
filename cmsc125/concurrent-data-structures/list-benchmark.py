#!/usr/bin/env python3
"""
Compare the single-lock linked list with the hand-over-hand linked list.

The list is pre-filled with --size keys. Each thread then does --ops
operations (lookups of random keys, plus --insert-pct percent inserts).
Time (seconds) for the threaded phase is reported against thread count.

Usage:
  python3 list-benchmark.py
  python3 list-benchmark.py --threads 1 2 4 8 --size 2000 --ops 5000
  python3 list-benchmark.py --insert-pct 20

Output: a table on the screen and list-benchmark.png next to this script.
Requires gcc and matplotlib (pip install matplotlib).
"""
import argparse, os, statistics, subprocess, tempfile

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

HERE = os.path.dirname(os.path.abspath(__file__))

ap = argparse.ArgumentParser(description="Single-lock vs hand-over-hand list benchmark")
ap.add_argument("--threads", type=int, nargs="+", default=[1, 2, 4],
                help="thread counts to test (default: 1 2 4)")
ap.add_argument("--size", type=int, default=1000,
                help="keys in the list before the test starts (default: 1000)")
ap.add_argument("--ops", type=int, default=5000,
                help="operations per thread (default: 5000)")
ap.add_argument("--insert-pct", type=int, default=0,
                help="percent of operations that are inserts, rest are lookups (default: 0)")
ap.add_argument("--runs", type=int, default=3,
                help="runs per data point; the median is used (default: 3)")
args = ap.parse_args()

KINDS = [("single", "Single lock", "x-"), ("hoh", "Hand-over-hand", "o-")]
cpus = len(os.sched_getaffinity(0))

# Build into a temp directory so no binary is left next to the sources.
tmpdir = tempfile.mkdtemp()
exe = os.path.join(tmpdir, "list-bench")
subprocess.run(["gcc", "-O2", "-Wall", "-pthread",
                os.path.join(HERE, "list-bench.c"), "-o", exe], check=True)

def timeit(kind, nthreads):
    """Median of several runs, to smooth out scheduling noise."""
    times = []
    for _ in range(args.runs):
        out = subprocess.run([exe, kind, str(nthreads), str(args.size),
                              str(args.ops), str(args.insert_pct)],
                             capture_output=True, text=True, check=True)
        times.append(float(out.stdout))
    return statistics.median(times)

print(f"CPUs available: {cpus}   list size: {args.size}   ops per thread: {args.ops}   "
      f"inserts: {args.insert_pct}%   runs (median): {args.runs}")
print(f"{'threads':>8}{'single lock (s)':>18}{'hand-over-hand (s)':>21}{'hoh / single':>15}")

results = {k: [] for k, _, _ in KINDS}
for n in args.threads:
    row = {k: timeit(k, n) for k, _, _ in KINDS}
    for k in row:
        results[k].append(row[k])
    ratio = row["hoh"] / row["single"]
    print(f"{n:>8}{row['single']:>18.4f}{row['hoh']:>21.4f}{ratio:>14.2f}x")

plt.figure(figsize=(6, 4))
for kind, label, style in KINDS:
    plt.plot(args.threads, results[kind], style, color="black", label=label,
             markerfacecolor="none" if style.startswith("o") else "black")
plt.xlabel("Threads")
plt.ylabel("Time (seconds)")
plt.xticks(args.threads)
plt.ylim(bottom=0)
plt.legend(frameon=True)
plt.title(f"Single-lock vs. Hand-over-hand Linked List\n"
          f"({args.size:,} nodes, {args.ops:,} ops per thread, "
          f"{args.insert_pct}% inserts, {cpus} CPU(s))", fontsize=10)
plt.tight_layout()
png = os.path.join(HERE, "list-benchmark.png")
plt.savefig(png, dpi=150)
print(f"\nSaved plot to {png}")
