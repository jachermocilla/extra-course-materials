/*
 * Ticket lock using fetch-and-add (OSTEP, Chapter 28: Locks) for x86-64.
 *
 * Build: gcc -O2 -Wall -pthread fetch-and-add.c -o fetch-and-add
 * Run:   ./fetch-and-add
 */
#include <stdio.h>
#include <pthread.h>

/* ---------------------------------------------------------------------
 * Atomic fetch-and-add
 *
 * Atomically adds 1 to *ptr and returns the OLD value of *ptr.
 *
 * x86-64 `xadd` loads the old memory value into the register operand
 * and stores the sum back to memory. The register starts out holding
 * the constant 1, since xadd has no immediate form. Unlike `xchg`, it is not
 * implicitly locked, so the `lock` prefix is required for atomicity
 * across cores.
 * ------------------------------------------------------------------- */
static inline int FetchAndAdd(volatile int *ptr) {
    int old = 1;                  /* amount to add */
    __asm__ __volatile__(
        "lock; xaddl %0, %1"
        : "+r"(old), "+m"(*ptr)
        :
        : "memory", "cc");
    return old;
}

/* ---------------------------------------------------------------------
 * Ticket lock interface from the book
 * ------------------------------------------------------------------- */
typedef struct __lock_t {
    volatile int ticket;          /* next ticket to hand out */
    volatile int turn;            /* ticket currently being served */
} lock_t;

void init(lock_t *lock) {
    lock->ticket = 0;
    lock->turn = 0;
}

void lock(lock_t *lock) {
    int myturn = FetchAndAdd(&lock->ticket);   /* take a ticket */
    while (lock->turn != myturn)
        ;                         /* spin-wait until it's our turn */
}

void unlock(lock_t *lock) {
    FetchAndAdd(&lock->turn);     /* serve the next ticket */
}

/* ---------------------------------------------------------------------
 * Test: several threads increment a shared counter under the lock.
 * ------------------------------------------------------------------- */
#define NTHREADS 8
#define NITERS   10000

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
