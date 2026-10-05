#!/usr/bin/env python3
"""
Compare test-and-set.c (spinning), yield.c (yielding), sleep.c (sleeping on a
futex), sleep-3-states.c (sleeping with a 3-state flag) and two-phase.c
(spin first, then sleep).

Builds each program at several thread counts, runs it, and prints one row
per run. See COLUMN_GUIDE below (also printed after the table) for what
each column means.

All runs are pinned to a single CPU core (children inherit the affinity), so
threads always compete for one core. Linux only (uses os.sched_setaffinity).

Usage: python3 compare.py [thread counts...]     e.g. python3 compare.py 2 4 8 16
Run it from the folder containing the five .c files.
"""
import os, re, resource, subprocess, sys, tempfile, time

HERE = os.path.dirname(os.path.abspath(__file__))

COLUMN_GUIDE = """
How to read the table
---------------------
case      Which lock was tested and how many threads were competing for it,
          e.g. "yield (8 thr)" = yield.c with 8 threads.

wall      Wall-clock time in seconds: how long the program took from start
          to finish, as measured by a stopwatch. Lower is faster.

user      CPU seconds spent running the program's own code in user mode.
          Spinning shows up here: a thread that loops waiting for a lock
          is "running", so it burns user time.

sys       CPU seconds spent inside the kernel on the program's behalf,
          mostly system calls (futex_wait / futex_wake / sched_yield).
          Locks that sleep show their cost here.

          Note: user + sys is the total CPU the lock used. A lock can
          have a short wall time but waste a lot of CPU, so compare both.

vol cs    Voluntary context switches: the thread itself gave up the CPU,
          usually because it blocked (went to sleep on a futex). A high
          number means threads are really sleeping while they wait.
          (sched_yield leaves the thread runnable, so Linux counts a
          yield as an involuntary switch, not a voluntary one.)

invol cs  Involuntary context switches: the scheduler took the CPU away
          because the thread's time slice ran out or another thread
          became runnable. A thread preempted while holding the lock
          makes everyone else wait longer.

Tips
----
* Compare rows with the same thread count against each other.
* Small numbers (e.g. 0.00 or 0.03) are close to the measurement noise;
  run several times and look for consistent differences.
* Pinned to one core, so threads must share the CPU. Results on a
  multi-core machine without pinning can look quite different.
"""

PROGRAMS = ["test-and-set", "yield", "sleep", "sleep-3-states", "two-phase"]
thread_counts = [int(a) for a in sys.argv[1:]] or [2, 8]

def run(exe):
    before = resource.getrusage(resource.RUSAGE_CHILDREN)
    start = time.time()
    subprocess.run([exe], stdout=subprocess.DEVNULL, timeout=300, check=True)
    wall = time.time() - start
    after = resource.getrusage(resource.RUSAGE_CHILDREN)
    return (wall,
            after.ru_utime - before.ru_utime,
            after.ru_stime - before.ru_stime,
            after.ru_nvcsw - before.ru_nvcsw,
            after.ru_nivcsw - before.ru_nivcsw)

# Pin this process (and every child it spawns) to one core.
core = min(os.sched_getaffinity(0))
os.sched_setaffinity(0, {core})
print(f"Pinned to CPU core {core} (machine has {os.cpu_count()} cores)")
print(f"{'case':30}{'wall':>8}{'user':>8}{'sys':>8}{'vol cs':>10}{'invol cs':>10}")
with tempfile.TemporaryDirectory() as tmp:
    for nt in thread_counts:
        for name in PROGRAMS:
            src = open(os.path.join(HERE, name + ".c")).read()
            src = re.sub(r"#define NTHREADS \d+", f"#define NTHREADS {nt}", src)
            c_file, exe = os.path.join(tmp, name + ".c"), os.path.join(tmp, name)
            open(c_file, "w").write(src)
            subprocess.run(["gcc", "-O2", "-pthread", c_file, "-o", exe], check=True)
            w, u, s, v, i = run(exe)
            label = f"{name} ({nt} thr)"
            print(f"{label:30}{w:8.2f}{u:8.2f}{s:8.2f}{v:10d}{i:10d}")

print(COLUMN_GUIDE)
