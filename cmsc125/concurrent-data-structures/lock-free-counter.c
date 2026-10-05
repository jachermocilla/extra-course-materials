/*
 * Concurrent counter without locks (OSTEP, Chapter 29) for x86-64.
 *
 * Build: gcc -O2 -Wall -pthread lock-free-counter.c -o lock-free-counter
 */
#include <stdio.h>
#include <pthread.h>

typedef struct __counter_t {
    int value;
} counter_t;

void init(counter_t *c) {
    c->value = 0;
}

void increment(counter_t *c) {
    c->value++;
}

void decrement(counter_t *c) {
    c->value--;
}

int get(counter_t *c) {
    return c->value;
}

/* Test: each thread does NITERS increments and NITERS/2 decrements. */
#define NTHREADS 8
#define NITERS   1000000

static counter_t counter;

static void *worker(void *arg) {
    (void)arg;
    for (int i = 0; i < NITERS; i++)
        increment(&counter);
    for (int i = 0; i < NITERS / 2; i++)
        decrement(&counter);
    return NULL;
}

int main(void) {
    pthread_t threads[NTHREADS];

    init(&counter);

    for (int i = 0; i < NTHREADS; i++)
        pthread_create(&threads[i], NULL, worker, NULL);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(threads[i], NULL);

    int expected = NTHREADS * (NITERS - NITERS / 2);
    printf("counter  = %d\n", get(&counter));
    printf("expected = %d\n", expected);
    printf("%s\n", get(&counter) == expected ? "PASS" : "FAIL (lost updates!)");
    return get(&counter) == expected ? 0 : 1;
}
