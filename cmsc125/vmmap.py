#!/usr/bin/env python3
"""
vmmap - explore the virtual address space of a Linux process, in the terminal.

Reads /proc/<pid>/maps and /proc/<pid>/smaps and renders:
  * a per-cluster "panorama" strip showing how regions pack together
  * a region table with log-scaled size bars, gaps between mappings collapsed
  * a category breakdown of virtual size vs resident size
  * an optional full-screen browser (--tui) for stepping through regions

Usage:
    vmmap.py <pid|self|name>        static snapshot
    vmmap.py <pid> --tui            interactive browser
    vmmap.py <pid> --watch 1.0      redraw every second
    vmmap.py --list                 list processes you can inspect

Reading another user's process needs matching privileges (root, or the same
uid plus a permissive /proc/sys/kernel/yama/ptrace_scope for smaps detail).
"""

from __future__ import annotations

import argparse
import math
import os
import re
import shutil
import sys
import textwrap
import time
from dataclasses import dataclass, field

PAGE = 4096

MAP_RE = re.compile(
    r"^([0-9a-fA-F]+)-([0-9a-fA-F]+)\s+"   # start-end
    r"(\S{4})\s+"                          # perms
    r"([0-9a-fA-F]+)\s+"                   # file offset
    r"(\S+)\s+"                            # dev
    r"(\d+)\s*"                            # inode
    r"(.*)$"                               # path
)

# ---------------------------------------------------------------------------
# categories
# ---------------------------------------------------------------------------

# key -> (label, 256-colour index)
CATEGORIES = {
    "code":   ("code",   210),
    "rodata": ("rodata", 216),
    "data":   ("data",   180),
    "heap":   ("heap",   185),
    "stack":  ("stack",  176),
    "lib":    ("lib",     74),
    "libtxt": ("lib.txt", 80),
    "file":   ("file",    72),
    "anon":   ("anon",   246),
    "kernel": ("kernel", 139),
    "guard":  ("guard",  238),
}

CATEGORY_ORDER = ["code", "rodata", "data", "heap", "stack",
                  "libtxt", "lib", "file", "anon", "kernel", "guard"]

DESCRIPTIONS = {
    "code":
        "Machine instructions of the program itself, mapped r-x straight from "
        "the executable file. Read-only so it can be shared: every process "
        "running this binary points at the same physical pages.",
    "rodata":
        "Read-only data from the executable — string literals, constants, "
        "jump tables, relocation info. Mapped r-- so a stray write faults "
        "instead of corrupting it.",
    "data":
        "Writable globals and statics from the executable (.data and .bss). "
        "Private and copy-on-write, so each process gets its own copy the "
        "first time it writes.",
    "heap":
        "The classic heap, grown and shrunk by brk()/sbrk() on behalf of "
        "malloc. Note that large allocations usually bypass this and come "
        "back as separate anonymous mappings instead.",
    "stack":
        "The main thread's stack. Grows downward toward lower addresses, and "
        "the kernel extends it automatically on fault. Threads get ordinary "
        "anonymous mappings rather than a region labelled like this.",
    "libtxt":
        "Executable code of a shared library. One physical copy in RAM is "
        "shared by every process that loaded the library, which is why "
        "resident size is usually far smaller than it looks.",
    "lib":
        "The non-executable parts of a shared library: its read-only data, "
        "the GOT/PLT the dynamic linker patches at load time, and its "
        "writable globals.",
    "file":
        "Some other file mapped into memory with mmap — a data file, a locale "
        "table, a memfd, or a System V shared memory segment. Pages are read "
        "from the file on demand.",
    "anon":
        "Memory backed by no file at all: large malloc arenas, thread stacks, "
        "and explicit mmap(MAP_ANONYMOUS). Handed out zero-filled on first "
        "touch, which is why virtual size often far exceeds resident size.",
    "kernel":
        "Pages the kernel maps into every process. [vdso] holds code for fast "
        "syscalls like gettimeofday, [vvar] the data that code reads, and "
        "[vsyscall] is a legacy version of the same idea.",
    "guard":
        "No permissions at all (---p). Address space deliberately reserved so "
        "that any access faults — used to pad the space between a library's "
        "segments and to catch stack overflow.",
}


# ---------------------------------------------------------------------------
# model
# ---------------------------------------------------------------------------

@dataclass
class Region:
    start: int
    end: int
    perms: str
    offset: int
    dev: str
    inode: int
    path: str
    detail: dict = field(default_factory=dict)   # smaps keys, in bytes
    category: str = "anon"

    @property
    def size(self) -> int:
        return self.end - self.start

    @property
    def rss(self):
        return self.detail.get("Rss")

    @property
    def pss(self):
        return self.detail.get("Pss")

    @property
    def swap(self):
        return self.detail.get("Swap")

    @property
    def dirty(self):
        if "Private_Dirty" not in self.detail and "Shared_Dirty" not in self.detail:
            return None
        return (self.detail.get("Private_Dirty", 0) +
                self.detail.get("Shared_Dirty", 0))

    @property
    def name(self) -> str:
        if self.path:
            return self.path
        if self.perms == "---p":
            return "(guard page)"
        return "(anonymous)"

    @property
    def short_name(self) -> str:
        n = self.name
        if n.startswith("/"):
            return os.path.basename(n)
        return n


@dataclass
class Snapshot:
    pid: int
    comm: str
    cmdline: str
    exe: str
    regions: list
    rollup: dict
    detailed: bool          # were we able to read smaps?
    taken_at: float

    @property
    def vsize(self) -> int:
        return sum(r.size for r in self.regions)

    @property
    def rss_total(self):
        if "Rss" in self.rollup:
            return self.rollup["Rss"]
        vals = [r.rss for r in self.regions if r.rss is not None]
        return sum(vals) if vals else None


# ---------------------------------------------------------------------------
# parsing
# ---------------------------------------------------------------------------

def read_text(path: str) -> str:
    with open(path, "r", errors="replace") as fh:
        return fh.read()


def parse_map_line(line: str):
    m = MAP_RE.match(line)
    if not m:
        return None
    start, end, perms, offset, dev, inode, path = m.groups()
    return Region(
        start=int(start, 16),
        end=int(end, 16),
        perms=perms,
        offset=int(offset, 16),
        dev=dev,
        inode=int(inode),
        path=path.strip(),
    )


def parse_maps(pid: int) -> list:
    out = []
    for line in read_text(f"/proc/{pid}/maps").splitlines():
        r = parse_map_line(line)
        if r:
            out.append(r)
    return out


def parse_smaps(pid: int) -> list:
    """Full smaps parse: map header lines interleaved with Key: N kB lines."""
    regions = []
    current = None
    with open(f"/proc/{pid}/smaps", "r", errors="replace") as fh:
        for line in fh:
            r = parse_map_line(line)
            if r is not None:
                current = r
                regions.append(current)
                continue
            if current is None:
                continue
            if ":" not in line:
                continue
            key, _, val = line.partition(":")
            val = val.strip()
            if val.endswith("kB"):
                try:
                    current.detail[key.strip()] = int(val[:-2].strip()) * 1024
                except ValueError:
                    pass
            elif key.strip() == "VmFlags":
                current.detail["VmFlags"] = val
    return regions


def parse_rollup(pid: int) -> dict:
    out = {}
    try:
        for line in read_text(f"/proc/{pid}/smaps_rollup").splitlines():
            if ":" not in line:
                continue
            key, _, val = line.partition(":")
            val = val.strip()
            if val.endswith("kB"):
                try:
                    out[key.strip()] = int(val[:-2].strip()) * 1024
                except ValueError:
                    pass
    except OSError:
        pass
    return out


def process_name(pid: int) -> str:
    try:
        return read_text(f"/proc/{pid}/comm").strip()
    except OSError:
        return "?"


def process_cmdline(pid: int) -> str:
    try:
        raw = read_text(f"/proc/{pid}/cmdline")
    except OSError:
        return ""
    parts = [p for p in raw.split("\0") if p]
    return " ".join(parts)


def process_exe(pid: int) -> str:
    try:
        return os.readlink(f"/proc/{pid}/exe")
    except OSError:
        return ""


def classify(r: Region, exe: str) -> str:
    p = r.path
    if p in ("[vdso]", "[vvar]", "[vsyscall]", "[vvar_vclock]", "[uprobes]"):
        return "kernel"
    if p == "[heap]":
        return "heap"
    if p.startswith("[stack"):
        return "stack"
    if r.perms == "---p" and not p:
        return "guard"
    if p and exe and p == exe:
        if "x" in r.perms:
            return "code"
        if r.perms.startswith("r--"):
            return "rodata"
        return "data"
    if p.startswith("/"):
        base = os.path.basename(p)
        if ".so" in base or base.startswith("ld-") or "/lib" in p:
            return "libtxt" if "x" in r.perms else "lib"
        return "file"
    if p.startswith("/memfd") or p.startswith("/dev/") or p.startswith("/SYSV"):
        return "file"
    return "anon"


def snapshot(pid: int) -> Snapshot:
    exe = process_exe(pid)
    detailed = True
    try:
        regions = parse_smaps(pid)
        if not regions:
            raise OSError("empty smaps")
    except OSError:
        detailed = False
        regions = parse_maps(pid)
    for r in regions:
        r.category = classify(r, exe)
    return Snapshot(
        pid=pid,
        comm=process_name(pid),
        cmdline=process_cmdline(pid),
        exe=exe,
        regions=regions,
        rollup=parse_rollup(pid),
        detailed=detailed,
        taken_at=time.time(),
    )


def resolve_target(token: str) -> int:
    if token == "self":
        return os.getpid()
    if token.isdigit():
        return int(token)
    matches = []
    for entry in os.listdir("/proc"):
        if not entry.isdigit():
            continue
        pid = int(entry)
        name = process_name(pid)
        if token.lower() in name.lower():
            matches.append((pid, name))
    if not matches:
        raise SystemExit(f"vmmap: no process matching {token!r}")
    if len(matches) > 1:
        listing = "\n".join(f"  {p:>7}  {n}" for p, n in matches[:20])
        raise SystemExit(
            f"vmmap: {token!r} matches {len(matches)} processes:\n{listing}\n"
            "Pass a pid instead."
        )
    return matches[0][0]


def list_processes():
    rows = []
    for entry in sorted(os.listdir("/proc"), key=lambda e: int(e) if e.isdigit() else 0):
        if not entry.isdigit():
            continue
        pid = int(entry)
        try:
            n = len(parse_maps(pid))
        except OSError:
            continue
        if n == 0:
            continue          # kernel thread: no address space of its own
        rows.append((pid, process_name(pid), n, process_cmdline(pid)[:60]))
    print(f"{'PID':>8}  {'REGIONS':>7}  NAME")
    for pid, name, n, cmd in rows:
        print(f"{pid:>8}  {n:>7}  {name}  {cmd}")


# ---------------------------------------------------------------------------
# formatting helpers
# ---------------------------------------------------------------------------

def human(n, precision=1) -> str:
    if n is None:
        return "-"
    if n < 1024:
        return f"{n} B"
    units = ["KiB", "MiB", "GiB", "TiB", "PiB"]
    v = float(n)
    for u in units:
        v /= 1024.0
        if v < 1024 or u == units[-1]:
            return f"{v:.{precision}f} {u}"
    return f"{v:.1f} PiB"


def compact(n) -> str:
    """Fixed-width-friendly size, e.g. 1.7M, 132K, 8.0T."""
    if n is None:
        return "-"
    if n == 0:
        return "0"
    units = [(1 << 50, "P"), (1 << 40, "T"), (1 << 30, "G"),
             (1 << 20, "M"), (1 << 10, "K")]
    for scale, suffix in units:
        if n >= scale:
            v = n / scale
            return f"{v:.0f}{suffix}" if v >= 10 else f"{v:.1f}{suffix}"
    return f"{n}"


def hexaddr(a: int, width: int = 12) -> str:
    return f"0x{a:0{width}x}"


BLOCKS = " ▏▎▍▌▋▊▉█"
SHADES = " ░▒▓█"


def bar(fraction: float, width: int) -> str:
    fraction = max(0.0, min(1.0, fraction))
    total = fraction * width
    full = int(total)
    rest = total - full
    s = "█" * full
    idx = int(rest * 8)
    if idx and len(s) < width:
        s += BLOCKS[idx]
    return s.ljust(width)


def log_fraction(size: int, biggest: int) -> float:
    if biggest <= 0:
        return 0.0
    lo = math.log2(PAGE)
    hi = math.log2(max(biggest, PAGE * 2))
    v = math.log2(max(size, PAGE))
    frac = (v - lo) / (hi - lo) if hi > lo else 1.0
    return max(frac, 0.02)


# ---------------------------------------------------------------------------
# colour
# ---------------------------------------------------------------------------

class Palette:
    def __init__(self, enabled: bool):
        self.enabled = enabled

    def fg(self, idx: int, text: str) -> str:
        if not self.enabled:
            return text
        return f"\033[38;5;{idx}m{text}\033[0m"

    def cat(self, key: str, text: str) -> str:
        return self.fg(CATEGORIES.get(key, ("", 245))[1], text)

    def dim(self, text: str) -> str:
        return f"\033[2m{text}\033[0m" if self.enabled else text

    def bold(self, text: str) -> str:
        return f"\033[1m{text}\033[0m" if self.enabled else text

    def rule(self, text: str) -> str:
        return self.fg(240, text)


# ---------------------------------------------------------------------------
# clustering + panorama
# ---------------------------------------------------------------------------

def cluster(regions: list, gap_threshold: int = 64 << 20) -> list:
    """Group regions separated by less than gap_threshold bytes."""
    clusters = []
    current = []
    for r in sorted(regions, key=lambda r: r.start):
        if current and r.start - current[-1].end > gap_threshold:
            clusters.append(current)
            current = []
        current.append(r)
    if current:
        clusters.append(current)
    return clusters


def panorama_line(regions: list, width: int, pal: Palette) -> str:
    """One strip of `width` cells covering [lo, hi) of this cluster."""
    lo = regions[0].start
    hi = regions[-1].end
    span = max(hi - lo, 1)
    per_cell = span / width

    coverage = [dict() for _ in range(width)]
    for r in regions:
        a = (r.start - lo) / per_cell
        b = (r.end - lo) / per_cell
        first = int(a)
        last = min(int(math.ceil(b)) - 1, width - 1)
        for c in range(max(first, 0), last + 1):
            overlap = min(b, c + 1) - max(a, c)
            if overlap > 0:
                coverage[c][r.category] = coverage[c].get(r.category, 0.0) + overlap

    out = []
    for cell in coverage:
        if not cell:
            out.append(pal.fg(236, "·"))
            continue
        cat = max(cell, key=cell.get)
        fill = min(sum(cell.values()), 1.0)
        ch = SHADES[min(int(fill * 4.0 + 0.999), 4)]
        out.append(pal.cat(cat, ch))
    return "".join(out)


# ---------------------------------------------------------------------------
# static rendering
# ---------------------------------------------------------------------------

def render_header(snap: Snapshot, pal: Palette, width: int) -> list:
    title = f"{snap.comm}  pid {snap.pid}"
    lines = [pal.bold(title)]
    if snap.cmdline:
        lines.append(pal.dim("  " + snap.cmdline[: width - 4]))

    rss = snap.rss_total
    facts = [
        f"{len(snap.regions)} regions",
        f"virtual {human(snap.vsize)}",
        f"resident {human(rss)}",
    ]
    if snap.rollup.get("Pss") is not None:
        facts.append(f"proportional {human(snap.rollup['Pss'])}")
    if snap.rollup.get("Swap"):
        facts.append(f"swapped {human(snap.rollup['Swap'])}")
    user = [r for r in snap.regions if r.category != "kernel"] or snap.regions
    lo = min(r.start for r in user)
    hi = max(r.end for r in user)
    facts.append(f"span {human(hi - lo, 0)}")
    ratio = (hi - lo) / max(snap.vsize, 1)
    facts.append(f"density 1 in {ratio:,.0f}")
    lines.append("  " + pal.rule(" · ").join(facts))
    if not snap.detailed:
        lines.append(pal.fg(214, "  smaps unreadable — sizes only, no residency"))
    return lines


def render_breakdown(snap: Snapshot, pal: Palette, width: int) -> list:
    totals = {}
    for r in snap.regions:
        v, res, cnt = totals.get(r.category, (0, 0, 0))
        totals[r.category] = (v + r.size, res + (r.rss or 0), cnt + 1)

    have_rss = snap.detailed
    key = (lambda kv: kv[1][1]) if have_rss else (lambda kv: kv[1][0])
    ordered = sorted(totals.items(), key=key, reverse=True)

    peak_v = max((t[0] for t in totals.values()), default=1)
    peak_r = max((t[1] for t in totals.values()), default=1) or 1
    bar_w = max(10, min(28, width - 46))

    lines = [pal.bold("Where the memory goes")]
    head = f"  {'':<8} {'count':>5} {'virtual':>9} {'resident':>9}"
    lines.append(pal.dim(head))
    for cat, (vsz, res, cnt) in ordered:
        label = CATEGORIES.get(cat, (cat, 245))[0]
        frac = (res / peak_r) if have_rss else (vsz / peak_v)
        b = pal.cat(cat, bar(frac, bar_w))
        lines.append(
            f"  {pal.cat(cat, label.ljust(8))} {cnt:>5} "
            f"{compact(vsz):>9} {compact(res) if have_rss else '-':>9}  {b}"
        )
    return lines


def render_regions(snap: Snapshot, pal: Palette, width: int,
                   show_gaps: bool, limit: int = None) -> list:
    regions = sorted(snap.regions, key=lambda r: r.start)
    biggest = max((r.size for r in regions), default=PAGE)
    bar_w = max(8, min(16, width - 66))
    name_w = max(16, width - (18 + 6 + 7 + 7 + bar_w + 10))

    lines = [pal.bold("Regions, low address to high")]
    lines.append(pal.dim(
        f"  {'address':<18} {'perm':<5} {'size':>6} {'res':>6} "
        f"{'':<{bar_w}}  region"
    ))

    prev_end = None
    count = 0
    for r in regions:
        if show_gaps and prev_end is not None and r.start > prev_end:
            gap = r.start - prev_end
            if gap >= PAGE:
                lines.append(pal.rule(
                    f"  {'':<18} {'':<5} {'':>6} {'':>6} "
                    f"{'':<{bar_w}}  ┄┄ {human(gap, 0)} unmapped ┄┄"
                ))
        prev_end = r.end

        b = pal.cat(r.category, bar(log_fraction(r.size, biggest), bar_w))
        perms = colour_perms(r.perms, pal)
        name = r.name
        if len(name) > name_w:
            name = "…" + name[-(name_w - 1):]
        lines.append(
            f"  {hexaddr(r.start):<18} {perms} {compact(r.size):>6} "
            f"{compact(r.rss) if snap.detailed else '-':>6} {b}  "
            f"{pal.cat(r.category, name)}"
        )
        count += 1
        if limit and count >= limit:
            lines.append(pal.dim(f"  … {len(regions) - count} more regions"))
            break
    return lines


def colour_perms(perms: str, pal: Palette) -> str:
    if not pal.enabled:
        return f"{perms:<5}"
    out = []
    for ch in perms:
        if ch == "-":
            out.append(pal.fg(238, ch))
        elif ch == "x":
            out.append(pal.fg(203, ch))
        elif ch == "w":
            out.append(pal.fg(215, ch))
        elif ch == "r":
            out.append(pal.fg(108, ch))
        else:
            out.append(pal.fg(245, ch))
    return "".join(out) + " "


def render_panorama(snap: Snapshot, pal: Palette, width: int) -> list:
    clusters = cluster(snap.regions)
    strip_w = max(20, width - 44)
    lines = [pal.bold("Clusters")]
    lines.append(pal.dim(
        "  each strip is one run of nearby mappings, drawn to its own scale"
    ))
    for group in clusters:
        lo = group[0].start
        hi = group[-1].end
        span = hi - lo
        mapped = sum(r.size for r in group)
        density = mapped / span if span else 1.0
        strip = panorama_line(group, strip_w, pal)
        plural = "s" if len(group) != 1 else " "
        stats = "{:>3} region{}  {:>9}  {:>4.0%} full".format(
            len(group), plural, human(span, 0), density)
        lines.append("  {:<18} {} {}".format(hexaddr(lo), strip, pal.dim(stats)))
    return lines


def render_legend(pal: Palette) -> list:
    parts = []
    for key in CATEGORY_ORDER:
        label, _ = CATEGORIES[key]
        parts.append(pal.cat(key, "█ " + label))
    return [pal.dim("Legend  ") + "  ".join(parts),
            pal.dim("        pass --explain for a description of each type")]


def render_glossary(snap: Snapshot, pal: Palette, width: int) -> list:
    """One paragraph per region type, restricted to types actually present."""
    present = {r.category for r in snap.regions}
    counts = {}
    for r in snap.regions:
        counts[r.category] = counts.get(r.category, 0) + 1

    lines = [pal.bold("What each region type is")]
    body_w = max(40, width - 14)
    for key in CATEGORY_ORDER:
        if key not in present:
            continue
        label = CATEGORIES[key][0]
        head = f"  {pal.cat(key, '█ ' + label.ljust(8))} {pal.dim(f'x{counts[key]}')}"
        lines.append(head)
        for chunk in textwrap.wrap(DESCRIPTIONS[key], body_w):
            lines.append("      " + chunk)
        lines.append("")
    if lines and lines[-1] == "":
        lines.pop()
    return lines


def render_static(snap: Snapshot, pal: Palette, width: int,
                  show_gaps: bool, limit: int, explain: bool = False) -> str:
    blocks = [
        render_header(snap, pal, width),
        render_panorama(snap, pal, width),
        render_breakdown(snap, pal, width),
        render_regions(snap, pal, width, show_gaps, limit),
        render_legend(pal),
    ]
    if explain:
        blocks.insert(-1, render_glossary(snap, pal, width))
    out = []
    for b in blocks:
        out.extend(b)
        out.append("")
    return "\n".join(out)


# ---------------------------------------------------------------------------
# interactive browser
# ---------------------------------------------------------------------------

def run_tui(pid: int, refresh: float):
    import curses

    def main(stdscr):
        curses.curs_set(0)
        stdscr.nodelay(True)
        use_colour = curses.has_colors()
        if use_colour:
            curses.start_color()
            curses.use_default_colors()
            for i, key in enumerate(CATEGORY_ORDER, start=1):
                idx = CATEGORIES[key][1]
                try:
                    curses.init_pair(i, idx if curses.COLORS > 8 else curses.COLOR_WHITE, -1)
                except curses.error:
                    pass
            try:
                curses.init_pair(30, 240, -1)   # rule
                curses.init_pair(31, 214, -1)   # warn
            except curses.error:
                pass

        pair_for = {key: i for i, key in enumerate(CATEGORY_ORDER, start=1)}

        def attr(key):
            if not use_colour:
                return curses.A_NORMAL
            return curses.color_pair(pair_for.get(key, 0))

        state = {
            "snap": snapshot(pid),
            "sel": 0,
            "top": 0,
            "sort": "address",
            "filter": "",
            "auto": refresh > 0,
            "help": False,
            "status": "",
            "last": time.time(),
        }

        def visible_regions():
            regs = state["snap"].regions
            f = state["filter"].lower()
            if f:
                regs = [r for r in regs
                        if f in r.name.lower() or f in r.category
                        or f in r.perms]
            if state["sort"] == "address":
                regs = sorted(regs, key=lambda r: r.start)
            elif state["sort"] == "size":
                regs = sorted(regs, key=lambda r: r.size, reverse=True)
            else:
                regs = sorted(regs, key=lambda r: (r.rss or 0), reverse=True)
            return regs

        def draw_help():
            stdscr.erase()
            h, w = stdscr.getmaxyx()
            body_w = max(30, w - 10)
            safe_addstr(stdscr, 0, 0, " vmmap — keys and region types",
                        curses.A_BOLD)
            y = 2
            for key, what in (
                ("↑ ↓ / j k", "move between regions"),
                ("PgUp PgDn", "move a screenful"),
                ("g / G", "jump to the lowest / highest address"),
                ("s", "cycle sort: address, size, resident"),
                ("/", "filter by name, category, or permissions"),
                ("r", "re-read /proc now"),
                ("a", "toggle auto-refresh"),
                ("? ", "this screen"),
                ("q", "quit"),
            ):
                safe_addstr(stdscr, y, 2, f"{key:<12} {what}"[: w - 3])
                y += 1

            y += 1
            present = {r.category for r in state["snap"].regions}
            for cat in CATEGORY_ORDER:
                if cat not in present or y >= h - 2:
                    continue
                label = CATEGORIES[cat][0]
                safe_addstr(stdscr, y, 2, ("█ " + label)[: w - 3], attr(cat))
                y += 1
                for chunk in textwrap.wrap(DESCRIPTIONS[cat], body_w):
                    if y >= h - 2:
                        break
                    safe_addstr(stdscr, y, 6, chunk[: w - 7])
                    y += 1
                y += 1
            safe_addstr(stdscr, h - 1, 0, " any key returns"[: w - 1],
                        attr_rule(use_colour))
            stdscr.refresh()

        def draw():
            if state["help"]:
                draw_help()
                return
            stdscr.erase()
            h, w = stdscr.getmaxyx()
            snap = state["snap"]
            regs = visible_regions()
            if not regs:
                state["sel"] = 0

            detail_h = 10
            list_top = 3
            list_h = max(3, h - list_top - detail_h - 1)

            # ---- header
            rss = snap.rss_total
            head = (f" {snap.comm}  pid {snap.pid} "
                    f"· {len(snap.regions)} regions "
                    f"· virtual {human(snap.vsize)} "
                    f"· resident {human(rss)}")
            safe_addstr(stdscr, 0, 0, head[: w - 1], curses.A_BOLD)
            sub = (f" sort {state['sort']} · filter "
                   f"{state['filter'] or '(none)'} · "
                   f"{'auto-refresh' if state['auto'] else 'paused'}"
                   f"   {state['status']}")
            safe_addstr(stdscr, 1, 0, sub[: w - 1], attr_rule(use_colour))
            safe_addstr(stdscr, 2, 0, "─" * (w - 1), attr_rule(use_colour))

            # ---- clamp selection & scroll
            state["sel"] = max(0, min(state["sel"], max(len(regs) - 1, 0)))
            if state["sel"] < state["top"]:
                state["top"] = state["sel"]
            if state["sel"] >= state["top"] + list_h:
                state["top"] = state["sel"] - list_h + 1
            state["top"] = max(0, min(state["top"], max(len(regs) - list_h, 0)))

            biggest = max((r.size for r in regs), default=PAGE)
            bar_w = max(6, min(14, w - 62))

            for i in range(list_h):
                idx = state["top"] + i
                if idx >= len(regs):
                    break
                r = regs[idx]
                selected = idx == state["sel"]
                y = list_top + i
                marker = "▶" if selected else " "
                b = bar(log_fraction(r.size, biggest), bar_w)
                name = r.short_name
                line = (f"{marker} {hexaddr(r.start):<18} {r.perms:<5} "
                        f"{compact(r.size):>6} "
                        f"{compact(r.rss) if snap.detailed else '-':>6} "
                        f"{b}  {name}")
                a = attr(r.category)
                if selected:
                    a |= curses.A_REVERSE | curses.A_BOLD
                safe_addstr(stdscr, y, 0, line[: w - 1], a)

            # ---- detail pane
            dy = list_top + list_h
            safe_addstr(stdscr, dy, 0, "─" * (w - 1), attr_rule(use_colour))
            if regs:
                r = regs[state["sel"]]
                info = [
                    f" {r.name}",
                    f" {hexaddr(r.start)} – {hexaddr(r.end)}   "
                    f"{human(r.size)}   {r.size // PAGE} pages   "
                    f"perms {r.perms}   offset {hexaddr(r.offset, 8)}",
                ]
                if snap.detailed:
                    d = r.detail
                    info.append(
                        f" resident {human(r.rss)}   proportional {human(r.pss)}   "
                        f"dirty {human(r.dirty)}   swap {human(r.swap)}"
                    )
                    info.append(
                        f" clean/dirty private "
                        f"{human(d.get('Private_Clean'))}/"
                        f"{human(d.get('Private_Dirty'))}   shared "
                        f"{human(d.get('Shared_Clean'))}/"
                        f"{human(d.get('Shared_Dirty'))}"
                    )
                    flags = d.get("VmFlags", "")
                    if flags:
                        info.append(f" flags {flags}")
                desc = DESCRIPTIONS.get(r.category, "")
                if desc:
                    label = CATEGORIES[r.category][0]
                    wrapped = textwrap.wrap(f"{label}: {desc}", max(30, w - 3))
                    info.extend(" " + line for line in wrapped[:2])
                for i, text in enumerate(info):
                    if dy + 1 + i < h - 1:
                        safe_addstr(stdscr, dy + 1 + i, 0, text[: w - 1],
                                    attr(r.category) if i == 0 else curses.A_NORMAL)

            keys = (" ↑↓ move  PgUp/PgDn page  g/G ends  s sort  "
                    "/ filter  r refresh  a auto  ? help  q quit")
            safe_addstr(stdscr, h - 1, 0, keys[: w - 1], attr_rule(use_colour))
            stdscr.refresh()

        def prompt(label):
            curses.echo()
            h, w = stdscr.getmaxyx()
            safe_addstr(stdscr, h - 1, 0, " " * (w - 1))
            safe_addstr(stdscr, h - 1, 0, label)
            stdscr.nodelay(False)
            try:
                value = stdscr.getstr(h - 1, len(label), 60).decode(errors="replace")
            except Exception:
                value = ""
            curses.noecho()
            stdscr.nodelay(True)
            return value.strip()

        draw()
        while True:
            try:
                ch = stdscr.getch()
            except KeyboardInterrupt:
                break

            regs = visible_regions()
            h, _ = stdscr.getmaxyx()
            page = max(1, h - 13)

            if ch == -1:
                if state["auto"] and time.time() - state["last"] >= max(refresh, 0.2):
                    try:
                        state["snap"] = snapshot(pid)
                        state["status"] = ""
                    except OSError:
                        state["status"] = "process gone"
                        state["auto"] = False
                    state["last"] = time.time()
                    draw()
                time.sleep(0.05)
                continue

            if state["help"]:
                state["help"] = False
                draw()
                continue

            if ch in (ord("q"), 27):
                break
            elif ch == ord("?"):
                state["help"] = True
            elif ch in (curses.KEY_DOWN, ord("j")):
                state["sel"] += 1
            elif ch in (curses.KEY_UP, ord("k")):
                state["sel"] -= 1
            elif ch == curses.KEY_NPAGE:
                state["sel"] += page
            elif ch == curses.KEY_PPAGE:
                state["sel"] -= page
            elif ch == ord("g"):
                state["sel"] = 0
            elif ch == ord("G"):
                state["sel"] = len(regs) - 1
            elif ch == ord("s"):
                order = ["address", "size", "resident"]
                state["sort"] = order[(order.index(state["sort"]) + 1) % 3]
            elif ch == ord("/"):
                state["filter"] = prompt("filter: ")
                state["sel"] = 0
                state["top"] = 0
            elif ch == ord("r"):
                try:
                    state["snap"] = snapshot(pid)
                    state["status"] = "refreshed"
                except OSError:
                    state["status"] = "process gone"
            elif ch == ord("a"):
                state["auto"] = not state["auto"]
            elif ch == curses.KEY_RESIZE:
                pass
            draw()

    def attr_rule(use_colour):
        import curses as _c
        return _c.color_pair(30) if use_colour else _c.A_DIM

    def safe_addstr(win, y, x, text, attr=0):
        try:
            win.addstr(y, x, text, attr)
        except Exception:
            pass

    curses.wrapper(main)


# ---------------------------------------------------------------------------
# entry point
# ---------------------------------------------------------------------------

def build_parser():
    p = argparse.ArgumentParser(
        prog="vmmap",
        description="Explore the virtual address space of a Linux process.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="examples:\n"
               "  vmmap.py self\n"
               "  vmmap.py 1234 --tui\n"
               "  vmmap.py bash --watch 0.5\n",
    )
    p.add_argument("target", nargs="?", help="pid, 'self', or a process name")
    p.add_argument("--tui", action="store_true", help="interactive browser")
    p.add_argument("--watch", type=float, metavar="SECONDS", default=0,
                   help="redraw every SECONDS")
    p.add_argument("--no-gaps", action="store_true",
                   help="hide the unmapped-gap rows")
    p.add_argument("--limit", type=int, default=0,
                   help="show at most N regions (0 = all)")
    p.add_argument("--explain", action="store_true",
                   help="describe each region type present in this process")
    p.add_argument("--no-color", action="store_true", help="disable colour")
    p.add_argument("--list", action="store_true",
                   help="list inspectable processes and exit")
    return p


def main(argv=None):
    if not os.path.isdir("/proc/self"):
        raise SystemExit("vmmap: needs a Linux /proc filesystem")

    args = build_parser().parse_args(argv)

    if args.list:
        list_processes()
        return 0

    if not args.target:
        build_parser().print_help()
        return 2

    pid = resolve_target(args.target)

    if not os.path.isdir(f"/proc/{pid}"):
        raise SystemExit(f"vmmap: no such process {pid}")

    if args.tui:
        run_tui(pid, args.watch or 1.0)
        return 0

    pal = Palette(enabled=not args.no_color and sys.stdout.isatty())
    limit = args.limit or None

    def once():
        width = shutil.get_terminal_size((100, 30)).columns
        try:
            snap = snapshot(pid)
        except PermissionError:
            raise SystemExit(f"vmmap: not allowed to read /proc/{pid}/maps")
        except FileNotFoundError:
            raise SystemExit(f"vmmap: process {pid} exited")
        return render_static(snap, pal, width, not args.no_gaps, limit,
                             explain=args.explain)

    if args.watch:
        try:
            while True:
                sys.stdout.write("\033[H\033[2J" if pal.enabled else "\n")
                print(once())
                time.sleep(args.watch)
        except KeyboardInterrupt:
            pass
    else:
        print(once())
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except BrokenPipeError:
        os._exit(0)
