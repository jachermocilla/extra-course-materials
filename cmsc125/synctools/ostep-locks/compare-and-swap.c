/*
 * Compare-and-swap spin lock (OSTEP, Chapter 28: Locks) for x86-64.
 *
 * Build: gcc -O2 -Wall -pthread compare-and-swap.c -o compare-and-swap
 * Run:   ./compare-and-swap
 */
#include <stdio.h>
#include <pthread.h>

/* ---------------------------------------------------------------------
 * Atomic compare-and-swap
 *
 * If *ptr == expected, atomically store `new_val` into *ptr.
 * Always returns the ORIGINAL value of *ptr (so the caller can tell
 * whether the swap happened: it did iff the return equals `expected`).
 *
 * x86-64 `lock cmpxchg` compares EAX with the memory operand; if equal
 * it stores the source register, otherwise it loads the memory value
 * into EAX. Either way EAX ends up holding the original memory value.
 * The `lock` prefix makes it atomic across cores and a full barrier.
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
 * Lock interface from the book
 * ------------------------------------------------------------------- */
typedef struct __lock_t {
    volatile int flag;            /* 0: lock is available, 1: lock is held */
} lock_t;

void init(lock_t *lock) {
    lock->flag = 0;               /* lock starts out available */
}

void lock(lock_t *lock) {
    /* Try to change flag 0 -> 1. If the old value was 1, someone else
     * holds the lock, so keep spinning. */
    while (CompareAndSwap(&lock->flag, 0, 1) == 1)
        ;                         /* spin-wait */
}

void unlock(lock_t *lock) {
    lock->flag = 0;               /* lock is available again */
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
