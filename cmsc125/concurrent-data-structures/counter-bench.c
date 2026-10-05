/*
 * Benchmark: precise (single lock) counter vs approximate counter.
 * Reproduces the experiment in OSTEP Chapter 29: each thread updates the
 * counter NITERS times, and we time how long all threads take.
 *
 * Build: gcc -O2 -Wall -pthread counter-bench.c -o counter-bench
 * Usage: ./counter-bench <precise|approximate> <nthreads> [niters] [threshold]
 * Prints the elapsed time in seconds.
 *
 * NUMCPUS is the number of per-CPU local counters in the approximate counter
 * (override with -DNUMCPUS=n). Any number of threads up to MAXTHREADS is
 * allowed; thread i uses local counter (i % NUMCPUS), so extra threads
 * simply share a local counter and its lock.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>

#ifndef NUMCPUS
#define NUMCPUS 4
#endif
#define MAXTHREADS 256

/* ---------------- precise counter: one lock ---------------- */
typedef struct {
    int value;
    pthread_mutex_t lock;
} precise_t;

static void precise_init(precise_t *c) {
    c->value = 0;
    pthread_mutex_init(&c->lock, NULL);
}

static void precise_increment(precise_t *c) {
    pthread_mutex_lock(&c->lock);
    c->value++;
    pthread_mutex_unlock(&c->lock);
}

static int precise_get(precise_t *c) {
    pthread_mutex_lock(&c->lock);
    int rc = c->value;
    pthread_mutex_unlock(&c->lock);
    return rc;
}

/* ---------------- approximate counter ---------------- */
typedef struct {
    int global;
    pthread_mutex_t glock;
    int local[NUMCPUS];
    pthread_mutex_t llock[NUMCPUS];
    int threshold;
} approx_t;

static void approx_init(approx_t *c, int threshold) {
    c->threshold = threshold;
    c->global = 0;
    pthread_mutex_init(&c->glock, NULL);
    for (int i = 0; i < NUMCPUS; i++) {
        c->local[i] = 0;
        pthread_mutex_init(&c->llock[i], NULL);
    }
}

static void approx_update(approx_t *c, int threadID, int amt) {
    pthread_mutex_lock(&c->llock[threadID]);
    c->local[threadID] += amt;
    if (c->local[threadID] >= c->threshold) {
        pthread_mutex_lock(&c->glock);
        c->global += c->local[threadID];
        pthread_mutex_unlock(&c->glock);
        c->local[threadID] = 0;
    }
    pthread_mutex_unlock(&c->llock[threadID]);
}

static int approx_get(approx_t *c) {
    pthread_mutex_lock(&c->glock);
    int val = c->global;
    pthread_mutex_unlock(&c->glock);
    return val;
}

/* ---------------- benchmark driver ---------------- */
static precise_t precise;
static approx_t approx;
static int use_approx = 0;
static int niters = 1000000;

static void *worker(void *arg) {
    int threadID = (int)(long)arg % NUMCPUS;
    if (use_approx)
        for (int i = 0; i < niters; i++)
            approx_update(&approx, threadID, 1);
    else
        for (int i = 0; i < niters; i++)
            precise_increment(&precise);
    return NULL;
}

static double now(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec + ts.tv_nsec / 1e9;
}

int main(int argc, char **argv) {
    if (argc < 3) {
        fprintf(stderr, "usage: %s <precise|approximate> <nthreads> [niters] [threshold]\n", argv[0]);
        return 2;
    }
    use_approx = strcmp(argv[1], "approximate") == 0;
    int nthreads = atoi(argv[2]);
    if (argc > 3) niters = atoi(argv[3]);
    int threshold = argc > 4 ? atoi(argv[4]) : 1024;
    if (nthreads < 1 || nthreads > MAXTHREADS) {
        fprintf(stderr, "nthreads must be 1..%d\n", MAXTHREADS);
        return 2;
    }

    precise_init(&precise);
    approx_init(&approx, threshold);

    pthread_t threads[MAXTHREADS];
    double start = now();
    for (long i = 0; i < nthreads; i++)
        pthread_create(&threads[i], NULL, worker, (void *)i);
    for (int i = 0; i < nthreads; i++)
        pthread_join(threads[i], NULL);
    double elapsed = now() - start;

    /* sanity check: precise must be exact; approximate may lag a little */
    long expected = (long)nthreads * niters;
    long got = use_approx ? approx_get(&approx) : precise_get(&precise);
    long max_lag = use_approx ? (long)NUMCPUS * (threshold - 1) : 0;
    if (got > expected || expected - got > max_lag) {
        fprintf(stderr, "wrong result: got %ld, expected %ld\n", got, expected);
        return 1;
    }

    printf("%.6f\n", elapsed);
    return 0;
}
