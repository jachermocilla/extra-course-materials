/*
 * Test-and-set lock with yield (OSTEP, Chapter 28: Locks) for x86-64.
 *
 * Instead of spinning while the lock is held, a waiting thread gives up
 * the CPU with sched_yield() so the lock holder can run and release it.
 *
 * Build: gcc -O2 -Wall -pthread yield.c -o yield
 * Run:   ./yield
 */
#include <stdio.h>
#include <pthread.h>
#include <sched.h>               /* sched_yield */

/* ---------------------------------------------------------------------
 * Atomic test-and-set
 *
 * Atomically stores `new_val` into *ptr and returns the OLD value.
 * On x86, `xchg` with a memory operand is implicitly locked (no `lock`
 * prefix needed), so it is atomic across cores and acts as a full
 * memory barrier.
 * ------------------------------------------------------------------- */
static inline int TestAndSet(volatile int *ptr, int new_val) {
    int old = new_val;
    __asm__ __volatile__(
        "xchgl %0, %1"
        : "+r"(old), "+m"(*ptr)   /* old is read/write, *ptr is read/write */
        :
        : "memory", "cc");        /* compiler barrier */
    return old;
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
    while (TestAndSet(&lock->flag, 1) == 1)
        sched_yield();            /* give up the CPU instead of spinning */
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
