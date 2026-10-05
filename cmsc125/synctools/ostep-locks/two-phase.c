/*
 * Two-phase lock using Linux futexes (x86-64).
 *
 * Phase 1: spin for a short while, hoping the lock holder releases the
 *          lock soon (cheap if the critical section is short).
 * Phase 2: if the lock is still held, go to sleep in the kernel with a
 *          futex, using the 3-state flag from sleep-3-states.c.
 *
 *   flag == 0   unlocked
 *   flag == 1   locked, no waiters
 *   flag == 2   locked, there may be sleeping waiters
 *
 * Build: gcc -O2 -Wall -pthread two-phase.c -o two-phase
 * Run:   ./two-phase
 */
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <linux/futex.h>

/* ---------------------------------------------------------------------
 * Atomic test-and-set (exchange): stores new_val into *ptr and returns
 * the OLD value. `xchg` with a memory operand is implicitly locked.
 * ------------------------------------------------------------------- */
static inline int TestAndSet(volatile int *ptr, int new_val) {
    int old = new_val;
    __asm__ __volatile__(
        "xchgl %0, %1"
        : "+r"(old), "+m"(*ptr)
        :
        : "memory", "cc");
    return old;
}

/* ---------------------------------------------------------------------
 * Atomic compare-and-swap: if *ptr == expected, store new_val.
 * Always returns the ORIGINAL value of *ptr.
 * ------------------------------------------------------------------- */
static inline int CompareAndSwap(volatile int *ptr, int expected, int new_val) {
    int original = expected;      /* goes in EAX */
    __asm__ __volatile__(
        "lock; cmpxchgl %2, %1"
        : "+a"(original), "+m"(*ptr)
        : "r"(new_val)
        : "memory", "cc");
    return original;
}

/* ---------------------------------------------------------------------
 * Futex wrappers (Linux)
 *
 * futex_wait: if *addr still equals `expected`, sleep until woken. The
 *             check and the sleep are atomic in the kernel, so a wake-up
 *             can't be lost between our check and going to sleep.
 * futex_wake: wake up to one thread sleeping on addr.
 * ------------------------------------------------------------------- */
static void futex_wait(volatile int *addr, int expected) {
    syscall(SYS_futex, addr, FUTEX_WAIT_PRIVATE, expected, NULL, NULL, 0);
}

static void futex_wake(volatile int *addr) {
    syscall(SYS_futex, addr, FUTEX_WAKE_PRIVATE, 1, NULL, NULL, 0);
}

/* ---------------------------------------------------------------------
 * Lock interface from the book
 * ------------------------------------------------------------------- */
typedef struct __lock_t {
    volatile int flag;            /* 0: unlocked, 1: locked, 2: locked + waiters */
} lock_t;

void init(lock_t *lock) {
    lock->flag = 0;               /* lock starts out available */
}

/* How many times to check the lock during the spin phase. */
#define SPIN_LIMIT 1000

void lock(lock_t *lock) {
    /* Phase 1: spin. Read the flag first and only attempt the atomic
     * operation when the lock looks free, to avoid hammering the cache
     * line with atomic instructions. */
    for (int i = 0; i < SPIN_LIMIT; i++) {
        if (lock->flag == 0 && CompareAndSwap(&lock->flag, 0, 1) == 0)
            return;               /* acquired while spinning */
    }

    /* Phase 2: sleep (same as sleep-3-states.c). Mark the lock "locked
     * with waiters" (2) and sleep until it is released. Each time we
     * wake up we set the state back to 2, because there may be other
     * sleepers. If the exchange returns 0, we own the lock. */
    int c = CompareAndSwap(&lock->flag, 0, 1);
    if (c != 0) {
        if (c != 2)
            c = TestAndSet(&lock->flag, 2);
        while (c != 0) {
            futex_wait(&lock->flag, 2);   /* sleep while flag == 2 */
            c = TestAndSet(&lock->flag, 2);
        }
    }
}

void unlock(lock_t *lock) {
    /* Atomically release the lock and look at the previous state. Only
     * if it was 2 can someone be asleep, so only then pay for a wake-up.
     * (This must be an atomic exchange, not a plain store, because we
     * need to know whether the flag was 2 at the moment of release.) */
    if (TestAndSet(&lock->flag, 0) == 2)
        futex_wake(&lock->flag);
}

/* ---------------------------------------------------------------------
 * Test: several threads increment a shared counter under the lock.
 * ------------------------------------------------------------------- */
#define NTHREADS 8
#define NITERS   1000000

static lock_t mutex;
static long counter = 0;

static void *worker(void *arg) {
    (void)arg;
    for (int i = 0; i < NITERS; i++) {
        lock(&mutex);
        counter++;                /* critical section */
        unlock(&mutex);
    }
    return NULL;
}

int main(void) {
    pthread_t threads[NTHREADS];

    init(&mutex);

    for (int i = 0; i < NTHREADS; i++)
        pthread_create(&threads[i], NULL, worker, NULL);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(threads[i], NULL);

    long expected = (long)NTHREADS * NITERS;
    printf("counter  = %ld\n", counter);
    printf("expected = %ld\n", expected);
    printf("%s\n", counter == expected ? "PASS" : "FAIL (race condition!)");
    return counter == expected ? 0 : 1;
}
