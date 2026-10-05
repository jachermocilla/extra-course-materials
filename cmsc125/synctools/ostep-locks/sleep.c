/*
 * Test-and-set lock with sleeping (OSTEP, Chapter 28: Locks) for x86-64.
 *
 * Instead of spinning or yielding, a waiting thread goes to sleep in the
 * kernel with the Linux futex system call. It uses no CPU while asleep,
 * and unlock() wakes one sleeper.
 *
 * Build: gcc -O2 -Wall -pthread sleep.c -o sleep
 * Run:   ./sleep
 */
#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/syscall.h>
#include <linux/futex.h>

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
 * Futex wrappers (Linux)
 *
 * futex_wait: if *addr still equals `expected`, put the caller to sleep
 *             until woken. The check and the sleep are atomic inside the
 *             kernel, so a wake-up between our test-and-set and the sleep
 *             is never lost (the call just returns immediately).
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
    volatile int flag;            /* 0: lock is available, 1: lock is held */
} lock_t;

void init(lock_t *lock) {
    lock->flag = 0;               /* lock starts out available */
}

void lock(lock_t *lock) {
    while (TestAndSet(&lock->flag, 1) == 1)
        futex_wait(&lock->flag, 1);   /* sleep until flag is no longer 1 */
}

void unlock(lock_t *lock) {
    lock->flag = 0;               /* lock is available again */
    futex_wake(&lock->flag);      /* wake one sleeping waiter, if any */
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
