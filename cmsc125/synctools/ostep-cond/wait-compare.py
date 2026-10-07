#!/usr/bin/env python3
"""
Compare the two ways a parent can wait for a child (OSTEP Chapter 30):
spinning on a flag (spin-based.c) vs. sleeping on a condition variable
(condition-variable.c).

The child "works" for a given delay (it sleeps, so it uses no CPU itself).
For each delay the table shows how long the parent waited (wall) and how
much CPU the process burned while waiting.

  wall   seconds until the parent noticed the child was done
  cpu    CPU seconds used by the process (user + sys). The child only
         sleeps, so this is the cost of waiting.
  cpu %  cpu / wall. ~100% = a core was kept fully busy; ~0% = parent
         was asleep.

Usage:  python3 wait-compare.py [--delays 10 100 500 1000] [--runs 3]
Run from the folder containing the .c files. Requires gcc.
"""
import argparse, os, statistics, subprocess, tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
ap = argparse.ArgumentParser()
ap.add_argument("--delays", type=int, nargs="+", default=[10, 100, 500, 1000],
                help="child work time in milliseconds (default: 10 100 500 1000)")
ap.add_argument("--runs", type=int, default=3, help="runs per data point (median)")
args = ap.parse_args()

tmp = tempfile.mkdtemp()
exe = os.path.join(tmp, "wait-bench")
subprocess.run(["gcc", "-O2", "-Wall", "-pthread",
                os.path.join(HERE, "wait-bench.c"), "-o", exe], check=True)

def measure(kind, delay):
    walls, cpus = [], []
    for _ in range(args.runs):
        out = subprocess.run([exe, kind, str(delay)], capture_output=True,
                             text=True, check=True).stdout.split()
        walls.append(float(out[0])); cpus.append(float(out[1]))
    return statistics.median(walls), statistics.median(cpus)

print(f"CPUs available: {len(os.sched_getaffinity(0))}   runs (median): {args.runs}")
print(f"{'child work':>11} | {'spin: wall':>10}{'cpu':>8}{'cpu %':>7} | "
      f"{'cond var: wall':>14}{'cpu':>8}{'cpu %':>7}")
for d in args.delays:
    sw, sc = measure("spin", d)
    cw, cc = measure("cond", d)
    print(f"{str(d)+' ms':>11} | {sw:>10.3f}{sc:>8.3f}{100*sc/sw:>6.0f}% | "
          f"{cw:>14.3f}{cc:>8.4f}{100*cc/cw:>6.1f}%")
print("\nThe child only sleeps, so all CPU time shown is the parent waiting.")
