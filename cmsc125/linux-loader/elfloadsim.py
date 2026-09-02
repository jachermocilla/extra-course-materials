#!/usr/bin/env python3
"""
elfload_sim.py -- simulate how the Linux kernel loads an ELF executable.

This mirrors the logic of fs/binfmt_elf.c:load_elf_binary() and the helpers it
calls (elf_map, padzero, set_brk, create_elf_tables, setup_arg_pages), plus the
architecture bits from arch/x86/ for x86-64 and arch/arm64/.

It does not execute anything.  It reads the file, decides where every byte
would land in the new address space, and prints the resulting memory map and
initial stack.

    ./elfload_sim.py /bin/ls -- -la /tmp
    ./elfload_sim.py --no-aslr --seed 1 ./a.out
    ./elfload_sim.py --json ./a.out
"""

from __future__ import annotations

import argparse
import json
import os
import random
import struct
import sys
from dataclasses import dataclass, field

# --------------------------------------------------------------------------
# ELF constants (include/uapi/linux/elf.h)
# --------------------------------------------------------------------------

ELFMAG = b"\x7fELF"

ELFCLASS32, ELFCLASS64 = 1, 2
ELFDATA2LSB, ELFDATA2MSB = 1, 2

ET_NONE, ET_REL, ET_EXEC, ET_DYN, ET_CORE = 0, 1, 2, 3, 4

PT_NULL = 0
PT_LOAD = 1
PT_DYNAMIC = 2
PT_INTERP = 3
PT_NOTE = 4
PT_SHLIB = 5
PT_PHDR = 6
PT_TLS = 7
PT_GNU_EH_FRAME = 0x6474E550
PT_GNU_STACK = 0x6474E551
PT_GNU_RELRO = 0x6474E552
PT_GNU_PROPERTY = 0x6474E553

PT_NAMES = {
    PT_NULL: "NULL", PT_LOAD: "LOAD", PT_DYNAMIC: "DYNAMIC", PT_INTERP: "INTERP",
    PT_NOTE: "NOTE", PT_SHLIB: "SHLIB", PT_PHDR: "PHDR", PT_TLS: "TLS",
    PT_GNU_EH_FRAME: "GNU_EH_FRAME", PT_GNU_STACK: "GNU_STACK",
    PT_GNU_RELRO: "GNU_RELRO", PT_GNU_PROPERTY: "GNU_PROPERTY",
}

PF_X, PF_W, PF_R = 1, 2, 4

EM_386, EM_X86_64, EM_ARM, EM_AARCH64, EM_RISCV = 3, 62, 40, 183, 243
EM_NAMES = {EM_386: "i386", EM_X86_64: "x86-64", EM_ARM: "arm",
            EM_AARCH64: "aarch64", EM_RISCV: "riscv"}

# auxv types (include/uapi/linux/auxvec.h + arch bits)
AT_NULL, AT_IGNORE, AT_EXECFD, AT_PHDR, AT_PHENT, AT_PHNUM = 0, 1, 2, 3, 4, 5
AT_PAGESZ, AT_BASE, AT_FLAGS, AT_ENTRY, AT_NOTELF, AT_UID = 6, 7, 8, 9, 10, 11
AT_EUID, AT_GID, AT_EGID, AT_PLATFORM, AT_HWCAP, AT_CLKTCK = 12, 13, 14, 15, 16, 17
AT_SECURE = 23
AT_BASE_PLATFORM = 24
AT_RANDOM = 25
AT_HWCAP2 = 26
AT_RSEQ_FEATURE_SIZE, AT_RSEQ_ALIGN = 27, 28
AT_EXECFN = 31
AT_SYSINFO, AT_SYSINFO_EHDR = 32, 33
AT_MINSIGSTKSZ = 51

AT_NAMES = {v: k for k, v in list(globals().items()) if k.startswith("AT_")}

PAGE_SIZE = 4096


def page_down(x: int, ps: int = PAGE_SIZE) -> int:
    return x & ~(ps - 1)


def page_up(x: int, ps: int = PAGE_SIZE) -> int:
    return (x + ps - 1) & ~(ps - 1)


# --------------------------------------------------------------------------
# ELF parsing.  Note: the kernel never looks at section headers.  Only the
# program headers matter for execution, so that is all we parse.
# --------------------------------------------------------------------------

@dataclass
class Phdr:
    p_type: int
    p_flags: int
    p_offset: int
    p_vaddr: int
    p_paddr: int
    p_filesz: int
    p_memsz: int
    p_align: int

    @property
    def type_name(self) -> str:
        return PT_NAMES.get(self.p_type, f"0x{self.p_type:08x}")

    @property
    def prot(self) -> str:
        return ("r" if self.p_flags & PF_R else "-") + \
               ("w" if self.p_flags & PF_W else "-") + \
               ("x" if self.p_flags & PF_X else "-")


class ElfError(Exception):
    pass


class Elf:
    def __init__(self, path: str, data: bytes | None = None):
        self.path = path
        self.data = data if data is not None else open(path, "rb").read()
        self._parse()

    # -- header ------------------------------------------------------------
    def _parse(self) -> None:
        d = self.data
        # The kernel reads BINPRM_BUF_SIZE (256) bytes and checks the magic in
        # elf_check_arch()/load_elf_binary().  A short file is -ENOEXEC.
        if len(d) < 64:
            raise ElfError("file too short to be an ELF")
        if d[:4] != ELFMAG:
            raise ElfError("bad magic: not an ELF file (kernel: -ENOEXEC)")

        self.ei_class = d[4]
        self.ei_data = d[5]
        self.ei_version = d[6]
        self.ei_osabi = d[7]

        if self.ei_class not in (ELFCLASS32, ELFCLASS64):
            raise ElfError(f"bad EI_CLASS {self.ei_class}")
        if self.ei_data not in (ELFDATA2LSB, ELFDATA2MSB):
            raise ElfError(f"bad EI_DATA {self.ei_data}")

        self.is64 = self.ei_class == ELFCLASS64
        self.wordsize = 8 if self.is64 else 4
        en = "<" if self.ei_data == ELFDATA2LSB else ">"
        self.endian = en

        if self.is64:
            fmt = en + "HHIQQQIHHHHHH"
            names = ("e_type e_machine e_version e_entry e_phoff e_shoff e_flags "
                     "e_ehsize e_phentsize e_phnum e_shentsize e_shnum e_shstrndx")
        else:
            fmt = en + "HHIIIIIHHHHHH"
            names = ("e_type e_machine e_version e_entry e_phoff e_shoff e_flags "
                     "e_ehsize e_phentsize e_phnum e_shentsize e_shnum e_shstrndx")
        vals = struct.unpack_from(fmt, d, 16)
        for name, val in zip(names.split(), vals):
            setattr(self, name, val)

        # -- program headers ----------------------------------------------
        # load_elf_phdrs(): sanity limits, then one read of the whole table.
        if self.e_phnum == 0:
            raise ElfError("no program headers (kernel: -ENOEXEC)")
        if self.e_phnum > 65536 // self.e_phentsize:
            raise ElfError("too many program headers")

        self.phdrs: list[Phdr] = []
        for i in range(self.e_phnum):
            off = self.e_phoff + i * self.e_phentsize
            if off + self.e_phentsize > len(d):
                raise ElfError("program header table runs past end of file")
            if self.is64:
                t, fl, o, va, pa, fs, ms, al = struct.unpack_from(en + "IIQQQQQQ", d, off)
            else:
                t, o, va, pa, fs, ms, fl, al = struct.unpack_from(en + "IIIIIIII", d, off)
            self.phdrs.append(Phdr(t, fl, o, va, pa, fs, ms, al))

    # -- convenience -------------------------------------------------------
    @property
    def loads(self) -> list[Phdr]:
        return [p for p in self.phdrs if p.p_type == PT_LOAD]

    @property
    def interp(self) -> str | None:
        for p in self.phdrs:
            if p.p_type == PT_INTERP:
                s = self.data[p.p_offset:p.p_offset + p.p_filesz]
                return s.split(b"\0", 1)[0].decode("utf-8", "replace")
        return None

    @property
    def gnu_stack(self) -> Phdr | None:
        for p in self.phdrs:
            if p.p_type == PT_GNU_STACK:
                return p
        return None

    @property
    def machine_name(self) -> str:
        return EM_NAMES.get(self.e_machine, f"EM_{self.e_machine}")

    def check_exec(self) -> None:
        """The gate at the top of load_elf_binary()."""
        if self.e_type not in (ET_EXEC, ET_DYN):
            raise ElfError(f"e_type is {self.e_type}, not ET_EXEC/ET_DYN "
                           "(kernel: -ENOEXEC)")
        if not self.loads:
            raise ElfError("no PT_LOAD segments (kernel: -ENOEXEC)")


# --------------------------------------------------------------------------
# Per-architecture parameters (the ELF_* macros in arch/*/include/asm/elf.h)
# --------------------------------------------------------------------------

@dataclass
class Arch:
    name: str
    task_size: int          # TASK_SIZE / DEFAULT_MAP_WINDOW
    stack_top: int          # STACK_TOP
    mmap_rnd_bits: int      # /proc/sys/vm/mmap_rnd_bits
    stack_rnd_mask: int     # STACK_RND_MASK, in pages
    brk_rnd_bits: int       # arch_randomize_brk entropy, in pages
    has_vdso: bool = True
    platform: str = ""

    @property
    def elf_et_dyn_base(self) -> int:
        # #define ELF_ET_DYN_BASE (DEFAULT_MAP_WINDOW / 3 * 2)
        return page_down(self.task_size // 3 * 2)


ARCHES = {
    EM_X86_64: Arch("x86-64", 0x7FFFFFFFF000, 0x7FFFFFFFF000,
                    mmap_rnd_bits=28, stack_rnd_mask=0x3FFFFF,
                    brk_rnd_bits=13, platform="x86_64"),
    EM_386: Arch("i386", 0xC0000000, 0xC0000000,
                 mmap_rnd_bits=8, stack_rnd_mask=0x7FF,
                 brk_rnd_bits=13, platform="i686"),
    EM_AARCH64: Arch("aarch64", 0x1000000000000, 0x1000000000000,
                     mmap_rnd_bits=18, stack_rnd_mask=0x3FFFFF,
                     brk_rnd_bits=13, platform="aarch64"),
    EM_ARM: Arch("arm", 0xC0000000, 0xC0000000,
                 mmap_rnd_bits=8, stack_rnd_mask=0x7FF,
                 brk_rnd_bits=13, platform="v7l"),
    EM_RISCV: Arch("riscv64", 0x4000000000, 0x4000000000,
                   mmap_rnd_bits=18, stack_rnd_mask=0x3FFFFF,
                   brk_rnd_bits=13, platform="riscv64"),
}


# --------------------------------------------------------------------------
# The simulated address space
# --------------------------------------------------------------------------

@dataclass
class Vma:
    start: int
    end: int
    prot: str            # "rwxp" style
    offset: int = 0
    backing: str = ""    # file path, "[stack]", "[heap]", "" for anon
    note: str = ""

    @property
    def size(self) -> int:
        return self.end - self.start


@dataclass
class Image:
    """Result of mapping one ELF object (the executable or the interpreter)."""
    name: str
    load_bias: int = 0
    entry: int = 0
    start_code: int = 0
    end_code: int = 0
    start_data: int = 0
    end_data: int = 0
    elf_bss: int = 0
    elf_brk: int = 0
    phdr_addr: int = 0
    vmas: list[Vma] = field(default_factory=list)


class Loader:
    def __init__(self, elf: Elf, argv: list[str], envp: list[str],
                 aslr: bool = True, seed: int | None = None,
                 interp_root: str = "/", secure: bool = False):
        self.elf = elf
        self.argv = argv
        self.envp = envp
        self.aslr = aslr
        self.secure = secure
        self.interp_root = interp_root
        self.rng = random.Random(seed)
        self.arch = ARCHES.get(elf.e_machine)
        if self.arch is None:
            raise ElfError(f"unsupported machine {elf.machine_name} "
                           "(kernel: elf_check_arch() fails, -ENOEXEC)")
        self.ws = elf.wordsize
        self.log: list[str] = []
        self.vmas: list[Vma] = []
        self.auxv: list[tuple[int, int]] = []
        self.stack_lines: list[tuple[int, str, str]] = []

    # -- helpers -----------------------------------------------------------
    def say(self, msg: str) -> None:
        self.log.append(msg)

    def rnd_pages(self, mask: int) -> int:
        return (self.rng.getrandbits(32) & mask) << 12 if self.aslr else 0

    def rnd_bits(self, bits: int) -> int:
        return (self.rng.getrandbits(bits) << 12) if self.aslr else 0

    def add(self, vma: Vma) -> None:
        if vma.end > vma.start:
            self.vmas.append(vma)

    # -- segment mapping (elf_map + the bss handling in load_elf_binary) ----
    def map_object(self, elf: Elf, name: str, load_bias: int,
                   backing: str, reserve_total: bool) -> Image:
        img = Image(name=name, load_bias=load_bias)
        loads = elf.loads
        first = True

        if reserve_total:
            # total_mapping_size(): for ET_DYN the kernel maps the whole span
            # once so the pieces stay contiguous, then overlays the segments.
            lo = page_down(loads[0].p_vaddr)
            hi = page_up(loads[-1].p_vaddr + loads[-1].p_memsz)
            self.say(f"  reserving {hi - lo:#x} bytes at "
                     f"{lo + load_bias:#x} for the whole image")

        for ph in loads:
            vaddr = ph.p_vaddr + load_bias
            # ELF_PAGESTART / ELF_PAGEOFFSET: mmap needs page-aligned args, so
            # the mapping is widened down to the containing page boundary and
            # the file offset is walked back by the same amount.
            page_off = vaddr - page_down(vaddr)
            map_start = page_down(vaddr)
            map_size = page_up(ph.p_filesz + page_off)
            file_off = ph.p_offset - page_off

            if ph.p_offset % PAGE_SIZE != vaddr % PAGE_SIZE:
                self.say(f"  !! p_offset and p_vaddr are not congruent mod page "
                         f"size in {ph.type_name} -- mmap cannot represent this")

            prot = ph.prot + "p"
            if map_size:
                self.add(Vma(map_start, map_start + map_size, prot,
                             file_off, backing,
                             f"PT_LOAD filesz={ph.p_filesz:#x}"))
                self.say(f"  mmap({map_start:#x}, {map_size:#x}, {ph.prot}, "
                         f"MAP_PRIVATE|MAP_FIXED, fd, {file_off:#x})")

            if first:
                first = False
                # phdr_addr: find the PT_LOAD that contains e_phoff.
                for p2 in loads:
                    if p2.p_offset <= elf.e_phoff < p2.p_offset + p2.p_filesz:
                        img.phdr_addr = elf.e_phoff - p2.p_offset + p2.p_vaddr + load_bias
                        break

            if ph.p_flags & PF_X:
                img.start_code = min(img.start_code or vaddr, vaddr)
                img.end_code = max(img.end_code, vaddr + ph.p_filesz)
            if not img.start_data:
                img.start_data = vaddr
            img.end_data = max(img.end_data, vaddr + ph.p_filesz)

            # .bss: everything between p_filesz and p_memsz must read as zero.
            if ph.p_memsz > ph.p_filesz:
                bss_start = vaddr + ph.p_filesz
                bss_end = vaddr + ph.p_memsz
                img.elf_bss = max(img.elf_bss, bss_start)
                img.elf_brk = max(img.elf_brk, bss_end)
                tail = page_up(bss_start) - bss_start
                if tail:
                    self.say(f"  padzero({bss_start:#x}): clear {tail:#x} bytes "
                             "in the tail of the last file page")
                if page_up(bss_end) > page_up(bss_start):
                    a, b = page_up(bss_start), page_up(bss_end)
                    self.add(Vma(a, b, prot, 0, "", "anonymous .bss"))
                    self.say(f"  vm_brk_flags({a:#x}, {b - a:#x})  # zero-fill "
                             "pages for .bss")
            else:
                img.elf_brk = max(img.elf_brk, vaddr + ph.p_memsz)

        img.entry = elf.e_entry + load_bias
        return img

    # -- the main event ----------------------------------------------------
    def run(self) -> None:
        elf = self.elf
        arch = self.arch
        elf.check_exec()

        self.say(f"binfmt_elf: {elf.machine_name} "
                 f"{'ET_EXEC' if elf.e_type == ET_EXEC else 'ET_DYN'}, "
                 f"{elf.e_phnum} program headers, entry {elf.e_entry:#x}")

        # 1. PT_INTERP -------------------------------------------------------
        interp_path = elf.interp
        interp_elf = None
        if interp_path:
            real = os.path.join(self.interp_root, interp_path.lstrip("/"))
            self.say(f"PT_INTERP = {interp_path}")
            try:
                interp_elf = Elf(real)
                interp_elf.check_exec()
            except (OSError, ElfError) as e:
                self.say(f"  !! cannot load interpreter: {e} "
                         "(kernel would fail the exec)")
                interp_elf = None

        # 2. Executable stack? ----------------------------------------------
        gs = elf.gnu_stack
        if gs is None:
            exec_stack = True
            self.say("no PT_GNU_STACK -> stack is executable (legacy default)")
        else:
            exec_stack = bool(gs.p_flags & PF_X)
            self.say(f"PT_GNU_STACK flags={gs.prot} -> stack is "
                     f"{'executable' if exec_stack else 'non-executable'}")

        # 3. Where does the executable go? ------------------------------------
        if elf.e_type == ET_EXEC:
            load_bias = 0
            self.say(f"ET_EXEC: fixed load address, load_bias = 0")
        else:
            if interp_elf is not None:
                # PIE with an interpreter: base at ELF_ET_DYN_BASE + randomness.
                load_bias = arch.elf_et_dyn_base + self.rnd_bits(arch.mmap_rnd_bits)
            else:
                # Static PIE: the kernel uses the mmap base instead.
                load_bias = arch.elf_et_dyn_base + self.rnd_bits(arch.mmap_rnd_bits)
            load_bias = page_down(load_bias - page_down(elf.loads[0].p_vaddr))
            self.say(f"ET_DYN: ELF_ET_DYN_BASE = {arch.elf_et_dyn_base:#x}, "
                     f"load_bias = {load_bias:#x}"
                     f"{'' if self.aslr else '  (ASLR off)'}")

        self.say("mapping the executable:")
        exe = self.map_object(elf, "exe", load_bias, self.elf.path,
                              reserve_total=(elf.e_type == ET_DYN))

        # 4. brk -------------------------------------------------------------
        # set_brk(): the heap starts at the page after the end of .bss, then
        # arch_randomize_brk() pushes it further away from the data segment.
        brk_base = page_up(exe.elf_brk or exe.end_data)
        brk_rnd = self.rnd_bits(arch.brk_rnd_bits)
        start_brk = brk_base + brk_rnd
        self.say(f"set_brk: mm->start_brk = mm->brk = {brk_base:#x}"
                 f"{f' + {brk_rnd:#x} random = {start_brk:#x}' if brk_rnd else ''}")
        self.add(Vma(start_brk, start_brk, "rw-p", 0, "[heap]",
                     "empty until the first brk()/sbrk()"))

        # 5. Interpreter ------------------------------------------------------
        interp_img = None
        interp_base = 0
        # mmap_base: the top-down mmap area sits below the stack, offset by a
        # gap for stack growth plus randomness.
        stack_gap = max(8 * 1024 * 1024, 128 * 1024 * 1024)  # RLIMIT_STACK-ish
        mmap_base = page_down(arch.stack_top - stack_gap -
                              self.rnd_bits(arch.mmap_rnd_bits))
        if interp_elf is not None:
            span = page_up(interp_elf.loads[-1].p_vaddr + interp_elf.loads[-1].p_memsz)
            if interp_elf.e_type == ET_DYN:
                interp_base = page_down(mmap_base - span)
            else:
                interp_base = 0
            self.say(f"mapping the interpreter at load_bias {interp_base:#x} "
                     f"(top-down from mmap_base {mmap_base:#x}):")
            interp_img = self.map_object(interp_elf, "interp", interp_base,
                                         interp_path, reserve_total=True)

        # 6. Entry point ------------------------------------------------------
        if interp_img is not None:
            entry = interp_img.entry
            self.say(f"entry point = interpreter entry {entry:#x} "
                     f"(the program's own entry {exe.entry:#x} is passed via "
                     "AT_ENTRY)")
        else:
            entry = exe.entry
            self.say(f"entry point = {entry:#x} (no interpreter)")

        # 7. vdso / vvar ------------------------------------------------------
        vdso_base = 0
        if arch.has_vdso:
            vdso_base = page_down(mmap_base - 0x100000 - self.rnd_bits(16))
            self.add(Vma(vdso_base - 3 * PAGE_SIZE, vdso_base, "r--p", 0,
                         "[vvar]", "kernel-shared time data"))
            self.add(Vma(vdso_base, vdso_base + 2 * PAGE_SIZE, "r-xp", 0,
                         "[vdso]", "virtual DSO"))
            self.say(f"arch_setup_additional_pages: vdso at {vdso_base:#x}")

        # 8. Stack ------------------------------------------------------------
        stack_top, sp = self.build_stack(exe, interp_base, vdso_base, exec_stack)

        self.exe = exe
        self.interp_img = interp_img
        self.entry = entry
        self.sp = sp
        self.stack_top = stack_top
        self.start_brk = start_brk
        self.mmap_base = mmap_base
        self.exec_stack = exec_stack
        self.vmas.sort(key=lambda v: v.start)

    # -- setup_arg_pages + create_elf_tables --------------------------------
    def build_stack(self, exe: Image, interp_base: int, vdso_base: int,
                    exec_stack: bool):
        arch = self.arch
        ws = self.ws
        pack = ("<" if self.elf.ei_data == ELFDATA2LSB else ">") + \
               ("Q" if ws == 8 else "I")

        # randomize_stack_top()
        rnd = self.rnd_pages(arch.stack_rnd_mask)
        stack_top = page_down(arch.stack_top - rnd)
        self.say(f"randomize_stack_top: stack top = {stack_top:#x} "
                 f"(STACK_TOP - {rnd:#x})")

        cursor = stack_top
        marks: list[tuple[int, int, str]] = []   # (start, end, label)

        def push_bytes(b: bytes, label: str) -> int:
            nonlocal cursor
            cursor -= len(b)
            marks.append((cursor, cursor + len(b), label))
            return cursor

        def push_str(s: str, label: str) -> int:
            return push_bytes(s.encode() + b"\0", label)

        # bprm->p starts one word below the top, then copy_strings() fills
        # downward: filename first, then envp (last to first), then argv.
        push_bytes(b"\0" * ws, "end marker (NULL)")
        execfn_addr = push_str(self.argv[0] if self.argv else self.elf.path,
                               "AT_EXECFN string")

        env_addrs = [0] * len(self.envp)
        for i in range(len(self.envp) - 1, -1, -1):
            env_addrs[i] = push_str(self.envp[i], f"envp[{i}]")

        arg_addrs = [0] * len(self.argv)
        for i in range(len(self.argv) - 1, -1, -1):
            arg_addrs[i] = push_str(self.argv[i], f"argv[{i}]")

        # create_elf_tables(): platform string, then 16 random bytes for
        # AT_RANDOM (glibc turns these into the stack canary and pointer guard).
        platform_addr = push_str(arch.platform, "AT_PLATFORM string") \
            if arch.platform else 0
        cursor &= ~(ws - 1)
        rand_addr = push_bytes(bytes(self.rng.getrandbits(8) for _ in range(16)),
                               "AT_RANDOM (16 bytes)")

        # auxv
        auxv: list[tuple[int, int]] = []
        if vdso_base:
            auxv.append((AT_SYSINFO_EHDR, vdso_base))
        auxv += [
            (AT_MINSIGSTKSZ, 3376),
            (AT_HWCAP, 0x178BFBFF),
            (AT_PAGESZ, PAGE_SIZE),
            (AT_CLKTCK, 100),
            (AT_PHDR, exe.phdr_addr),
            (AT_PHENT, self.elf.e_phentsize),
            (AT_PHNUM, self.elf.e_phnum),
            (AT_BASE, interp_base),
            (AT_FLAGS, 0),
            (AT_ENTRY, exe.entry),
            (AT_UID, os.getuid()), (AT_EUID, os.geteuid()),
            (AT_GID, os.getgid()), (AT_EGID, os.getegid()),
            (AT_SECURE, 1 if self.secure else 0),
            (AT_RANDOM, rand_addr),
            (AT_HWCAP2, 0),
        ]
        if platform_addr:
            auxv.append((AT_PLATFORM, platform_addr))
        auxv.append((AT_EXECFN, execfn_addr))
        auxv.append((AT_NULL, 0))
        self.auxv = auxv

        # The vector block: argc, argv[], NULL, envp[], NULL, auxv[], and the
        # whole thing has to leave sp 16-byte aligned at entry (STACK_ROUND).
        items = 1 + len(self.argv) + 1 + len(self.envp) + 1 + 2 * len(auxv)
        sp = (cursor - items * ws) & ~0xF
        self.say(f"create_elf_tables: {items} words of vectors, "
                 f"initial sp = {sp:#x} (16-byte aligned)")

        p = sp
        marks.append((p, p + ws, f"argc = {len(self.argv)}"))
        p += ws
        for i, a in enumerate(arg_addrs):
            marks.append((p, p + ws, f"argv[{i}] -> {a:#x}"))
            p += ws
        marks.append((p, p + ws, "argv NULL"))
        p += ws
        for i, a in enumerate(env_addrs):
            marks.append((p, p + ws, f"envp[{i}] -> {a:#x}"))
            p += ws
        marks.append((p, p + ws, "envp NULL"))
        p += ws
        for t, v in auxv:
            name = AT_NAMES.get(t, f"AT_{t}")
            marks.append((p, p + 2 * ws, f"{name} = {v:#x}"))
            p += 2 * ws

        # The VMA itself: one page below sp is mapped, and it grows down.
        stack_start = page_down(sp) - PAGE_SIZE
        prot = "rw" + ("x" if exec_stack else "-") + "p"
        self.add(Vma(stack_start, stack_top, prot, 0, "[stack]",
                     "grows down, guard gap of 256 pages below"))

        self.stack_marks = sorted(marks, reverse=True)
        return stack_top, sp

    # -- output -------------------------------------------------------------
    def print_report(self, show_stack: bool = True) -> None:
        w = 12 if self.ws == 8 else 8
        print("=" * 78)
        print(f"loading {self.elf.path}")
        print("=" * 78)
        for line in self.log:
            print(line)

        print()
        print("resulting address space (as /proc/self/maps would show it)")
        print("-" * 78)
        prev_end = None
        for v in self.vmas:
            if prev_end is not None and v.start > prev_end:
                gap = v.start - prev_end
                print(f"{'':>{2*w+1}}   {'':4}  {'':>10}   ... {gap:#x} bytes unmapped ...")
            size = v.size
            print(f"{v.start:0{w}x}-{v.end:0{w}x} {v.prot} {v.offset:08x} "
                  f"{size:>10,}  {v.backing or '[anon]'}"
                  + (f"   # {v.note}" if v.note else ""))
            prev_end = v.end

        print()
        print("register / mm state handed to the new program")
        print("-" * 78)
        print(f"  entry (rip/pc)   {self.entry:#x}")
        print(f"  sp               {self.sp:#x}")
        print(f"  mm->start_code   {self.exe.start_code:#x}")
        print(f"  mm->end_code     {self.exe.end_code:#x}")
        print(f"  mm->start_data   {self.exe.start_data:#x}")
        print(f"  mm->end_data     {self.exe.end_data:#x}")
        print(f"  mm->start_brk    {self.start_brk:#x}")
        print(f"  mm->start_stack  {self.sp:#x}")
        print(f"  mm->mmap_base    {self.mmap_base:#x}")
        print(f"  stack            {'executable' if self.exec_stack else 'NX'}")

        if show_stack:
            print()
            print("initial stack, top of memory first")
            print("-" * 78)
            for start, end, label in self.stack_marks:
                print(f"  {start:0{w}x}  {end - start:>3} B  {label}")
            print(f"  {'':{w}}         <- rsp points at argc")

    def to_json(self) -> str:
        return json.dumps({
            "path": self.elf.path,
            "machine": self.elf.machine_name,
            "type": "ET_EXEC" if self.elf.e_type == ET_EXEC else "ET_DYN",
            "interp": self.elf.interp,
            "entry": self.entry,
            "sp": self.sp,
            "load_bias": self.exe.load_bias,
            "interp_base": self.interp_img.load_bias if self.interp_img else None,
            "start_brk": self.start_brk,
            "exec_stack": self.exec_stack,
            "maps": [{"start": v.start, "end": v.end, "prot": v.prot,
                      "offset": v.offset, "backing": v.backing} for v in self.vmas],
            "auxv": [{"type": AT_NAMES.get(t, str(t)), "value": v}
                     for t, v in self.auxv],
        }, indent=2)


# --------------------------------------------------------------------------
# A quick static dump of the program headers, for orientation
# --------------------------------------------------------------------------

def print_phdrs(elf: Elf) -> None:
    print("program headers (what the kernel actually reads)")
    print("-" * 78)
    print(f"{'type':<14} {'offset':>10} {'vaddr':>14} {'filesz':>10} "
          f"{'memsz':>10} {'flg':>4} {'align':>8}")
    for p in elf.phdrs:
        print(f"{p.type_name:<14} {p.p_offset:>#10x} {p.p_vaddr:>#14x} "
              f"{p.p_filesz:>#10x} {p.p_memsz:>#10x} {p.prot:>4} "
              f"{p.p_align:>#8x}")
    print()


def main() -> int:
    ap = argparse.ArgumentParser(
        description="Simulate how the Linux kernel loads an ELF binary.")
    ap.add_argument("binary")
    ap.add_argument("args", nargs="*", help="argv[1:] for the simulated process")
    ap.add_argument("--no-aslr", action="store_true",
                    help="behave as with /proc/sys/kernel/randomize_va_space = 0")
    ap.add_argument("--seed", type=int, default=None,
                    help="seed the randomness so runs are reproducible")
    ap.add_argument("--env", action="append", default=None,
                    metavar="K=V", help="environment entry (repeatable)")
    ap.add_argument("--sysroot", default="/",
                    help="prefix for resolving PT_INTERP")
    ap.add_argument("--secure", action="store_true",
                    help="simulate a setuid exec (AT_SECURE=1)")
    ap.add_argument("--phdrs", action="store_true",
                    help="also dump the program header table")
    ap.add_argument("--no-stack", action="store_true",
                    help="skip the stack dump")
    ap.add_argument("--json", action="store_true")
    opts = ap.parse_args()

    try:
        elf = Elf(opts.binary)
    except (OSError, ElfError) as e:
        print(f"error: {e}", file=sys.stderr)
        return 1

    if opts.phdrs:
        print_phdrs(elf)

    env = opts.env if opts.env is not None else [
        "PATH=/usr/local/bin:/usr/bin:/bin",
        "HOME=/root",
        "LANG=C.UTF-8",
    ]

    loader = Loader(elf,
                    argv=[opts.binary] + opts.args,
                    envp=env,
                    aslr=not opts.no_aslr,
                    seed=opts.seed,
                    interp_root=opts.sysroot,
                    secure=opts.secure)
    try:
        loader.run()
    except ElfError as e:
        print(f"exec would fail: {e}", file=sys.stderr)
        return 1

    if opts.json:
        print(loader.to_json())
    else:
        loader.print_report(show_stack=not opts.no_stack)
    return 0


if __name__ == "__main__":
    sys.exit(main())
