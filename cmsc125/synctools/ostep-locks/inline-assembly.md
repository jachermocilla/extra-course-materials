# Inline Assembly in GCC

GCC's inline assembly (extended `asm`) lets you embed machine instructions in C while telling the compiler exactly how they touch your variables, so it can still optimize around them.

## Syntax

```c
__asm__ __volatile__(
    "template"            // assembly instructions
    : outputs             // C variables the asm writes
    : inputs              // C values the asm reads
    : clobbers            // other things the asm changes
);
```

The sections are positional. Empty ones can be omitted from the end, but you keep the colons if a later section is used (e.g. `::: "memory"` for no outputs or inputs).

## The Template

The template is a string of instructions in AT&T syntax by default, so the source comes before the destination. Operands are referenced as `%0`, `%1`, `%2`, numbered in order across outputs then inputs. A literal register name needs `%%` (e.g. `%%eax`), but you rarely write registers directly, because the compiler picks them for you.

## Constraints

Constraints describe each operand:

| Constraint | Meaning |
|---|---|
| `"r"` | any general-purpose register |
| `"m"` | a memory operand |
| `"a"` | specifically `EAX`/`RAX` (also `"b"`, `"c"`, `"d"`, `"S"`, `"D"`) |
| `"i"` | an immediate constant |

Modifiers go in front of the constraint:

| Modifier | Meaning |
|---|---|
| `=` | write-only output |
| `+` | read-write operand |

## Clobbers

Clobbers list what the asm changes beyond its declared operands:

- `"cc"` means the condition flags are modified.
- `"memory"` means the asm reads or writes memory in ways the compiler can't see, so it acts as a compiler barrier and prevents reordering memory accesses across it.
- A register name such as `"rax"` means that register is trashed.

## `volatile`

`volatile` tells the compiler not to delete the asm or move it around, even if its outputs look unused. Use it for anything with side effects.

## Examples from the Lock Implementations

### Test-and-set with `xchg`

```c
static inline int TestAndSet(volatile int *ptr, int new_val) {
    int old = new_val;
    __asm__ __volatile__(
        "xchgl %0, %1"
        : "+r"(old), "+m"(*ptr)
        :
        : "memory", "cc");
    return old;
}
```

`%0` is `old`, held in a register and both read and written. `%1` is the memory at `*ptr`, also read and written. `xchgl` swaps them atomically. With a memory operand, `xchg` is implicitly locked, so no `lock` prefix is needed. The `"memory"` clobber stops the compiler from moving other loads and stores across the swap, which matters for a lock.

### Compare-and-swap with `cmpxchg`

```c
static inline int CompareAndSwap(volatile int *ptr, int expected, int new_val) {
    int original = expected;      /* goes in EAX */
    __asm__ __volatile__(
        "lock; cmpxchgl %2, %1"
        : "+a"(original), "+m"(*ptr)
        : "r"(new_val)
        : "memory", "cc");
    return original;
}
```

`cmpxchg` implicitly compares against and returns through `EAX`, so the `"+a"` constraint pins `original` to that register. If `*ptr` equals `EAX`, the instruction stores `new_val` into `*ptr`. Otherwise it loads the memory value into `EAX`. Either way `original` ends up holding the old value of `*ptr`. The `lock` prefix makes it atomic across cores.

### Fetch-and-add with `xadd`

```c
static inline int FetchAndAdd(volatile int *ptr, int value) {
    int old = value;
    __asm__ __volatile__(
        "lock; xaddl %0, %1"
        : "+r"(old), "+m"(*ptr)
        :
        : "memory", "cc");
    return old;
}
```

`xaddl` loads the old value of `*ptr` into the register and stores the sum back to memory in one step. Unlike `xchg`, it is not implicitly locked, so the `lock` prefix is required.

## Practical Tips

- Lying to the compiler in the constraints causes subtle, optimization-dependent bugs. If the asm touches it, declare it.
- Inline asm isn't portable. For atomics, modern code usually uses the `__atomic_*` builtins or C11 `<stdatomic.h>`, which compile to the same instructions on any architecture. Hand-written asm like this is best for learning what's happening underneath.
- Check your work with `gcc -O2 -S` to see the assembly the compiler generated around your block.
