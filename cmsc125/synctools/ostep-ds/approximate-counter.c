/*
 * Approximate (sloppy) counter (OSTEP, Chapter 29), using pthread_mutex.
 *
 * Each CPU has a local count with its own lock. Threads normally update
 * only their local count. When a local count reaches `threshold`, it is
 * transferred to the global count (protected by the global lock).
 *
 * Build: gcc -O2 -Wall -pthread approximate-counter.c -o approximate-counter
 */
#include <stdio.h>
#include <pthread.h>

#define NUMCPUS 4

typedef struct __counter_t {
    int global;                       // global count
    pthread_mutex_t glock;            // global lock
    int local[NUMCPUS];               // local count (per cpu)
    pthread_mutex_t llock[NUMCPUS];   // ... and locks
    int threshold;                    // update frequency
} counter_t;

// init: record threshold, init locks, init values
//       of all local counts and global count
void init(counter_t *c, int threshold) {
    c->threshold = threshold;

    c->global = 0;
    pthread_mutex_init(&c->glock, NULL);

    int i;
    for (i = 0; i < NUMCPUS; i++) {
        c->local[i] = 0;
        pthread_mutex_init(&c->llock[i], NULL);
    }
}

// update: usually, just grab local lock and update local amount
//         once local count has risen by 'threshold', grab global
//         lock and transfer local values to it
void update(counter_t *c, int threadID, int amt) {
    pthread_mutex_lock(&c->llock[threadID]);
    c->local[threadID] += amt;                  // assumes amt > 0
    if (c->local[threadID] >= c->threshold) {   // transfer to global
        pthread_mutex_lock(&c->glock);
        c->global += c->local[threadID];
        pthread_mutex_unlock(&c->glock);
        c->local[threadID] = 0;
    }
    pthread_mutex_unlock(&c->llock[threadID]);
}

// get: just return global amount (which may not be perfect)
int get(counter_t *c) {
    pthread_mutex_lock(&c->glock);
    int val = c->global;
    pthread_mutex_unlock(&c->glock);
    return val;                                 // only approximate!
}

/* ---------------------------------------------------------------------
 * Test: NTHREADS threads each add 1 to the counter NITERS times. Thread i
 * uses local counter (i % NUMCPUS). After all threads finish, the global
 * count may lag the true total by at most NUMCPUS * (threshold - 1),
 * because each local count holds fewer than `threshold` unflushed updates.
 * ------------------------------------------------------------------- */
#define NTHREADS  8
#define NITERS    1000000
#define THRESHOLD 1024

static counter_t counter;

static void *worker(void *arg) {
    int threadID = (int)(long)arg % NUMCPUS;
    for (int i = 0; i < NITERS; i++)
        update(&counter, threadID, 1);
    return NULL;
}

int main(void) {
    pthread_t threads[NTHREADS];

    init(&counter, THRESHOLD);

    for (long i = 0; i < NTHREADS; i++)
        pthread_create(&threads[i], NULL, worker, (void *)i);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(threads[i], NULL);

    int expected = NTHREADS * NITERS;
    int got = get(&counter);
    int max_error = NUMCPUS * (THRESHOLD - 1);

    printf("true total   = %d\n", expected);
    printf("get()        = %d\n", got);
    printf("difference   = %d (allowed at most %d)\n", expected - got, max_error);

    int ok = got <= expected && expected - got <= max_error;
    printf("%s\n", ok ? "PASS" : "FAIL");
    return ok ? 0 : 1;
}
