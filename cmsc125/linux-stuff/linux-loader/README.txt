# elfload_sim.py — what really happens when you type `./a.out`

A small Python program that reads a Linux executable and shows you, step by
step, how the kernel would load it into memory. Nothing is executed. The
program only reads bytes from the file and prints where each one would end up.

It is written for people taking an OS, systems programming, or compilers
course who have heard the words "ELF", "loader", and "ASLR" and want to see
what they actually mean.

---

## 1. The question this answers

You compile a C program and run it:

```
$ gcc hello.c -o hello
$ ./hello
```

Between pressing Enter and `main()` running, the kernel does a surprising
amount of work. The shell calls `fork()` and then `execve("./hello", ...)`.
Inside `execve`, the kernel:

1. opens the file and looks at the first few bytes,
2. recognises the ELF format and hands the file to the ELF loader,
3. throws away the old program's entire address space,
4. maps pieces of the new file into memory at particular addresses,
5. builds a fresh stack containing the command-line arguments and environment,
6. jumps to the new program's first instruction.

This tool simulates steps 2 through 6 and prints the result.

---

## 2. Vocabulary you need

**Virtual address space.** Every process believes it has the machine's whole
memory to itself, addressed from 0 upward. The kernel and the CPU's MMU
maintain that illusion. An address like `0x555555554000` is a *virtual*
address; where it lives in physical RAM is not the program's business.

**Page.** Memory is managed in 4 KiB chunks called pages. Every address the
kernel maps must be a multiple of 4096. This single constraint explains a lot
of the loader's odd-looking arithmetic.

**Mapping.** `mmap()` tells the kernel "make the region of virtual memory
starting at address A, of length L, contain the contents of file F starting at
offset O, with permissions P". Nothing is copied. The pages are filled in
lazily on first access (a page fault). Loading a program is mostly just a
handful of `mmap()` calls.

**VMA (virtual memory area).** The kernel's record of one such mapping: a
start, an end, permissions, and what backs it. `cat /proc/self/maps` prints
the VMAs of the process reading it. Our tool prints its simulated VMAs in the
same format.

**ELF.** *Executable and Linkable Format*, the file format for programs,
shared libraries, and object files on Linux. An ELF file has two tables of
contents:

- the **section header table** — used by the compiler, assembler, linker, and
  debugger. Sections are things like `.text`, `.data`, `.rodata`, `.symtab`.
- the **program header table** — used by the *loader*. Its entries are called
  *segments*, and each one says "map this range of the file at this virtual
  address with these permissions".

A crucial fact that surprises most people: **the kernel never reads section
headers.** You can delete the entire section header table and the program
still runs. Only program headers matter at execution time. This tool
deliberately parses only program headers, for the same reason.

---

## 3. Installing and running

You need Python 3.10 or newer. There are no dependencies.

```
$ chmod +x elfload_sim.py
$ ./elfload_sim.py /bin/ls
```

Useful flags:

| flag | what it does |
| --- | --- |
| `--phdrs` | also print the program header table, like `readelf -l` |
| `--no-aslr` | pretend `/proc/sys/kernel/randomize_va_space` is 0 |
| `--seed N` | fix the randomness so two runs match |
| `--env K=V` | set one environment variable (repeatable) |
| `--sysroot DIR` | look for the dynamic linker under `DIR` instead of `/` |
| `--no-stack` | skip the stack dump |
| `--json` | machine-readable output |

Arguments after the binary become the simulated `argv`:

```
$ ./elfload_sim.py /bin/ls -- -la /tmp
```

**Start with `--no-aslr`.** Randomised addresses change every run and make it
hard to see the pattern. With ASLR off you get the same tidy addresses you see
under gdb.

---

## 4. Reading the output

Here is `./elfload_sim.py --no-aslr /bin/ls`, annotated.

### 4a. The header

```
binfmt_elf: x86-64 ET_DYN, 13 program headers, entry 0x6d30
PT_INTERP = /lib64/ld-linux-x86-64.so.2
PT_GNU_STACK flags=rw- -> stack is non-executable
```

`ET_DYN` means this is a *position-independent executable* (PIE). Modern
distributions compile everything this way so that the load address can be
randomised. The older `ET_EXEC` type demands a fixed address baked in at link
time.

`PT_INTERP` names the **dynamic linker**. Because `ls` uses shared libraries,
the kernel will not jump to `ls` directly. It loads a second program — the
dynamic linker, `ld-linux-x86-64.so.2` — and jumps there instead. The linker
loads libc and friends, applies relocations, then jumps to `ls`.

`PT_GNU_STACK` is a segment that maps nothing. Its permission bits exist only
to tell the kernel whether the stack should be executable. Since it lacks the
`x` bit, the stack is marked no-execute, which is what stops the classic
"inject shellcode onto the stack" attack.

### 4b. Choosing where to put the program

```
ET_DYN: ELF_ET_DYN_BASE = 0x555555554000, load_bias = 0x555555554000  (ASLR off)
```

A PIE's program headers say "load me at address 0" — a placeholder. The kernel
picks a real base and adds it to every address in the file. That offset is the
**load bias**. With ASLR off it is a fixed constant, `TASK_SIZE / 3 * 2`
rounded down to a page, which is where the famous `0x5555...` addresses in gdb
come from. With ASLR on, up to 28 bits of randomness are added.

### 4c. Mapping the segments

```
  mmap(0x555555554000, 0x4000,  r--, MAP_PRIVATE|MAP_FIXED, fd, 0x0)
  mmap(0x555555558000, 0x15000, r-x, MAP_PRIVATE|MAP_FIXED, fd, 0x4000)
  mmap(0x55555556d000, 0x8000,  r--, MAP_PRIVATE|MAP_FIXED, fd, 0x19000)
  mmap(0x555555575000, 0x3000,  rw-, MAP_PRIVATE|MAP_FIXED, fd, 0x20000)
```

Four `PT_LOAD` segments, four `mmap()` calls. Notice the permissions: read-only
metadata, read+execute code, read-only constants, read+write data. Nothing is
both writable and executable — the W^X rule.

`MAP_PRIVATE` means writes go to a copy-on-write page and never reach the file
on disk, which is why two copies of `ls` can share one physical copy of the
code but each get their own globals.

### 4d. The page-alignment trick

Segments in the file are not page-aligned; `mmap` demands that they are. The
loader handles this by rounding the address *down* to a page boundary and
pulling the file offset back by the same amount:

```
map_start = page_down(vaddr)
page_off  = vaddr - map_start
file_off  = p_offset - page_off
map_size  = page_up(p_filesz + page_off)
```

The extra bytes dragged in at the front are harmless — they belong to the
previous segment and get overwritten or ignored. This is why the linker takes
care to make `p_offset ≡ p_vaddr (mod 4096)` for every loadable segment: it
guarantees that one `mmap` can satisfy both constraints at once.

### 4e. Zeroing the `.bss`

```
  padzero(0x555555577278): clear 0xd88 bytes in the tail of the last file page
  vm_brk_flags(0x555555578000, 0x1000)  # zero-fill pages for .bss
```

A `PT_LOAD` segment has two sizes: `p_filesz` (bytes present in the file) and
`p_memsz` (bytes the program expects in memory). When `p_memsz > p_filesz`, the
difference is `.bss` — uninitialised globals, which the C standard says must
read as zero. Storing megabytes of zeroes in the file would be wasteful, so the
loader creates them at load time in two steps:

1. `padzero()` writes zeroes over the remainder of the last page that came from
   the file (that page contains real data at the start and junk after it),
2. any whole pages beyond that get a fresh anonymous mapping, which the kernel
   guarantees is zero-filled.

That anonymous region is the `[anon]` line in the memory map.

### 4f. The heap

```
set_brk: mm->start_brk = mm->brk = 0x555555579000
```

The heap begins immediately after `.bss` and is initially **empty** — zero
bytes long. It grows when the program calls `brk()`/`sbrk()`, which is what
`malloc()` does for small allocations. With ASLR on, the kernel also pushes the
start of the heap a random distance away from the data segment, so that
overflowing a global cannot reliably reach heap metadata.

### 4g. The dynamic linker

```
mapping the interpreter at load_bias 0x7ffff7fc5000 (top-down from mmap_base 0x7ffff7fff000)
...
entry point = interpreter entry 0x7ffff7fe4540 (the program's own entry
              0x55555555ad30 is passed via AT_ENTRY)
```

The same mapping procedure runs a second time for `ld-linux`, placed near the
top of the address space where `mmap()` hands out regions. The kernel then sets
the CPU's instruction pointer to the *linker's* entry point, not the program's.
The program's real entry address is handed over as an `AT_ENTRY` auxiliary
vector entry, and the linker jumps there once it has finished its work.

A statically linked binary has no `PT_INTERP`, so this whole step is skipped
and control goes straight to the program. Try it:

```
$ gcc -static hello.c -o hello_static
$ ./elfload_sim.py --no-aslr --phdrs hello_static
```

### 4h. The memory map

```
555555554000-555555558000 r--p 00000000     16,384  /bin/ls
555555558000-55555556d000 r-xp 00004000     86,016  /bin/ls
55555556d000-555555575000 r--p 00019000     32,768  /bin/ls
555555575000-555555578000 rw-p 00020000     12,288  /bin/ls
555555578000-555555579000 rw-p 00000000      4,096  [anon]      # .bss
                                 ... 0x2aaaa2983000 bytes unmapped ...
7ffff7efc000-7ffff7eff000 r--p 00000000     12,288  [vvar]
7ffff7eff000-7ffff7f01000 r-xp 00000000      8,192  [vdso]
7ffff7fc5000-7ffff7fc6000 r--p 00000000      4,096  /lib64/ld-linux-x86-64.so.2
7ffff7fc6000-7ffff7ff1000 r-xp 00001000    176,128  /lib64/ld-linux-x86-64.so.2
7ffff7ff1000-7ffff7ffb000 r--p 0002c000     40,960  /lib64/ld-linux-x86-64.so.2
7ffff7ffb000-7ffff7fff000 rw-p 00036000     16,384  /lib64/ld-linux-x86-64.so.2
7fffffffd000-7ffffffff000 rw-p 00000000      8,192  [stack]
```

The columns match `/proc/self/maps`: address range, permissions, file offset,
size, and what backs the mapping. The gaps are enormous — a 47-bit address
space is mostly holes, and touching a hole gives you a segmentation fault.

`[vdso]` is a tiny shared library the kernel injects into every process. It
implements calls like `gettimeofday()` without the cost of entering the kernel.
`[vvar]` is the read-only data page it reads the clock from.

Compare our simulated output with reality:

```
$ setarch -R cat /proc/self/maps
```

(`setarch -R` disables ASLR for one command.) The addresses should look
strikingly familiar.

### 4i. The initial stack

This is the part textbooks usually skip. Before jumping to the program, the
kernel *writes data onto the new stack*: the argument strings, the environment
strings, and arrays of pointers to them. Then, at the very bottom, it leaves
the stack pointer.

Reading the dump from high addresses downward:

```
  7ffdc5baefe1    3 B  argv[1]                  <- the actual strings
  7ffdc5baefd7   10 B  argv[0]
  7ffdc5baefd0    7 B  AT_PLATFORM string
  7ffdc5baefc0   16 B  AT_RANDOM (16 bytes)
  ...
  7ffdc5baee70   16 B  AT_SYSINFO_EHDR = 0x7fd953919000     <- auxiliary vector
  7ffdc5baee68    8 B  envp NULL
  7ffdc5baee60    8 B  envp[0] -> 0x7ffdc5baefe4            <- pointer arrays
  7ffdc5baee58    8 B  argv NULL
  7ffdc5baee50    8 B  argv[1] -> 0x7ffdc5baefe1
  7ffdc5baee48    8 B  argv[0] -> 0x7ffdc5baefd7
  7ffdc5baee40    8 B  argc = 2
                       <- rsp points at argc
```

So `argc` and `argv` are not passed in registers. They are simply sitting on
the stack when the program starts, and the C runtime (`_start` in libc) picks
them up and calls `main(argc, argv)`.

Above the pointer arrays sits the **auxiliary vector**, a list of
(key, value) pairs the kernel uses to tell the dynamic linker things it cannot
work out for itself:

| entry | meaning |
| --- | --- |
| `AT_PHDR`, `AT_PHNUM` | where the program's own program headers ended up |
| `AT_ENTRY` | the program's real entry point |
| `AT_BASE` | where the dynamic linker itself was loaded |
| `AT_PAGESZ` | page size |
| `AT_RANDOM` | 16 fresh random bytes — glibc turns these into the stack canary |
| `AT_SECURE` | 1 if this was a setuid exec, which makes libc ignore `LD_PRELOAD` |
| `AT_SYSINFO_EHDR` | address of the vdso |

You can see the real thing on your own machine:

```
$ LD_SHOW_AUXV=1 /bin/true
```

Finally, `sp` is 16-byte aligned. The x86-64 ABI requires it, and SSE
instructions fault if it is not.

---

## 5. Things to try

1. Run with and without `--no-aslr` several times. Which regions move
   independently? (Answer: the executable, the linker/mmap region, the stack,
   and the heap all get separate random offsets.)
2. Compile with `gcc -no-pie` and compare against a PIE build. Where does the
   program land, and what happens to `load_bias`?
3. Declare `static char buf[1 << 20];` in a C file. Does the binary grow by a
   megabyte? Watch what `vm_brk_flags` does in the output, then check with
   `ls -l`.
4. Build with `-static` and with `-static-pie`, and explain why the second one
   still has a load bias but no interpreter.
5. Add `-z execstack` at link time and see `PT_GNU_STACK` change.
6. Try `./elfload_sim.py --phdrs` on an aarch64 or riscv64 binary from another
   machine. The tool has per-architecture parameters, so cross-inspection works.
7. Read the ELF header of a `.o` file with the tool. It should refuse — why is
   `ET_REL` not executable?

---

## 6. How this maps to the real kernel

Every step corresponds to code in `fs/binfmt_elf.c`, which is about 1500 lines
and is worth reading once you have the shape in your head.

| this tool | kernel |
| --- | --- |
| `Elf._parse` | `load_elf_phdrs()` |
| `Loader.run` | `load_elf_binary()` |
| `Loader.map_object` | `elf_map()` and the `PT_LOAD` loop |
| bss handling | `padzero()`, `vm_brk_flags()` |
| `start_brk` | `set_brk()`, `arch_randomize_brk()` |
| stack construction | `setup_arg_pages()`, `create_elf_tables()` |
| `Arch` table | `ELF_ET_DYN_BASE`, `STACK_TOP`, `STACK_RND_MASK` in `arch/*/include/asm/elf.h` |

Reference reading: `man 5 elf`, `man 7 mmap`, `man 3 getauxval`, and the
System V ABI x86-64 supplement (chapter 3.4 covers the initial process state).

---

## 7. What this does *not* do

It is a teaching model, not an emulator. Specifically:

- **No execution.** No instructions are interpreted, no relocations applied, no
  shared libraries beyond the interpreter are loaded. Everything libc does
  after `_start` is out of scope.
- **`AT_HWCAP` and the vdso size are hardcoded**, because they depend on the
  CPU and the running kernel rather than on the file.
- **Randomisation uses Python's PRNG**, seeded for reproducibility. Real ASLR
  draws from the kernel's entropy pool, and the exact number of bits depends on
  `/proc/sys/vm/mmap_rnd_bits`.
- **`MAP_FIXED_NOREPLACE`, memory-tagging, CET/shadow stacks, and `PT_TLS`**
  are noted but not modelled.
- **Only `PT_INTERP` chaining is handled**, not `#!` scripts, `binfmt_misc`, or
  core dumps.

If a real `/proc/PID/maps` disagrees with the simulated one in some interesting
way, that discrepancy is usually the most educational thing in the whole
exercise — go find out why.
