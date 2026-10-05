#!/usr/bin/env python3
"""
Reproduce the OSTEP Chapter 29 figure "Performance of Traditional vs.
Approximate Counters": time (seconds) vs. number of threads, where each
thread updates the counter 1,000,000 times (approximate threshold = 1024).

Usage:
  python3 counter-benchmark.py                          # threads 1 2 3 4
  python3 counter-benchmark.py --threads 1 2 4 8 16 32
  python3 counter-benchmark.py --threshold 4096 --niters 500000 --runs 5

The approximate counter gets one local counter per available CPU (NUMCPUS),
so with more threads than CPUs, several threads share a local counter.

Output: a table on the screen and counter-benchmark.png next to this script.

Requires gcc and matplotlib (pip install matplotlib).
"""
import argparse, os, statistics, subprocess, tempfile

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

HERE = os.path.dirname(os.path.abspath(__file__))
ap = argparse.ArgumentParser(description="Precise vs approximate counter benchmark")
ap.add_argument("--threads", type=int, nargs="+", default=[1, 2, 3, 4],
                help="thread counts to test (default: 1 2 3 4)")
ap.add_argument("--threshold", type=int, default=1024,
                help="approximate counter threshold S (default: 1024)")
ap.add_argument("--niters", type=int, default=1000000,
                help="updates per thread (default: 1000000)")
ap.add_argument("--runs", type=int, default=3,
                help="runs per data point; the median is used (default: 3)")
args = ap.parse_args()
threshold, niters, runs, THREADS = args.threshold, args.niters, args.runs, args.threads
KINDS = ["precise", "approximate"]
cpus = len(os.sched_getaffinity(0))

# Build into a temp directory so no binary is left next to the sources.
tmpdir = tempfile.mkdtemp()
exe = os.path.join(tmpdir, "counter-bench")
subprocess.run(["gcc", "-O2", "-Wall", "-pthread", f"-DNUMCPUS={cpus}",
                os.path.join(HERE, "counter-bench.c"), "-o", exe], check=True)

def timeit(kind, nthreads):
    """Median of several runs, to smooth out scheduling noise."""
    times = []
    for _ in range(runs):
        out = subprocess.run([exe, kind, str(nthreads), str(niters), str(threshold)],
                             capture_output=True, text=True, check=True)
        times.append(float(out.stdout))
    return statistics.median(times)

print(f"CPUs available: {cpus}   threshold: {threshold}   "
      f"iterations per thread: {niters}   runs (median): {runs}")
print(f"{'threads':>8}{'precise (s)':>14}{'approximate (s)':>18}")

results = {k: [] for k in KINDS}
for n in THREADS:
    row = {k: timeit(k, n) for k in KINDS}
    for k in KINDS:
        results[k].append(row[k])
    print(f"{n:>8}{row['precise']:>14.4f}{row['approximate']:>18.4f}")

plt.figure(figsize=(6, 4))
plt.plot(THREADS, results["precise"], "x-", color="black", label="Precise")
plt.plot(THREADS, results["approximate"], "o-", color="black",
         markerfacecolor="none", label="Approximate")
plt.xlabel("Threads")
plt.ylabel("Time (seconds)")
plt.xticks(THREADS)
plt.ylim(bottom=0)
plt.legend(frameon=True)
plt.title(f"Traditional vs. Approximate Counters\n"
          f"(threshold S = {threshold}, {niters:,} updates per thread, {cpus} CPU(s))",
          fontsize=10)
plt.tight_layout()
png = os.path.join(HERE, "counter-benchmark.png")
plt.savefig(png, dpi=150)
print(f"\nSaved plot to {png}")
