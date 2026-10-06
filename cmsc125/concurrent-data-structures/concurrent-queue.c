/*
 * Concurrent queue with two locks (OSTEP, Chapter 29), using pthread_mutex.
 * This is the Michael and Scott queue: one lock for the head (dequeue)
 * and one for the tail (enqueue), plus a dummy node so that enqueuers
 * and dequeuers never touch the same pointers and can run in parallel.
 *
 * Build: gcc -O2 -Wall -pthread concurrent-queue.c -o concurrent-queue
 */
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <sched.h>
#include <pthread.h>

typedef struct __node_t {
    int value;
    struct __node_t *next;
} node_t;

typedef struct __queue_t {
    node_t *head;
    node_t *tail;
    pthread_mutex_t headLock;
    pthread_mutex_t tailLock;
} queue_t;

void Queue_Init(queue_t *q) {
    node_t *tmp = malloc(sizeof(node_t));
    tmp->next = NULL;
    q->head = q->tail = tmp;
    pthread_mutex_init(&q->headLock, NULL);
    pthread_mutex_init(&q->tailLock, NULL);
}

void Queue_Enqueue(queue_t *q, int value) {
    node_t *tmp = malloc(sizeof(node_t));
    assert(tmp != NULL);

    tmp->value = value;
    tmp->next = NULL;

    pthread_mutex_lock(&q->tailLock);
    q->tail->next = tmp;
    q->tail = tmp;
    pthread_mutex_unlock(&q->tailLock);
}

int Queue_Dequeue(queue_t *q, int *value) {
    pthread_mutex_lock(&q->headLock);
    node_t *tmp = q->head;
    node_t *newHead = tmp->next;
    if (newHead == NULL) {
        pthread_mutex_unlock(&q->headLock);
        return -1; // queue was empty
    }
    *value = newHead->value;
    q->head = newHead;
    pthread_mutex_unlock(&q->headLock);
    free(tmp);
    return 0;
}

/* ---------------------------------------------------------------------
 * Test: NPRODUCERS threads each enqueue NITEMS unique values while
 * NCONSUMERS threads dequeue concurrently. Checks that every value is
 * dequeued exactly once (none lost, none duplicated) and that each
 * consumer sees the values of any one producer in the order that
 * producer enqueued them (FIFO).
 * ------------------------------------------------------------------- */
#define NPRODUCERS 4
#define NCONSUMERS 4
#define NITEMS     10000
#define TOTAL      (NPRODUCERS * NITEMS)

static queue_t queue;
static int seen[TOTAL];           // how many times each value was dequeued
static int consumed = 0;          // total values dequeued so far
static int order_errors = 0;

static void *producer(void *arg) {
    long p = (long)arg;
    for (int i = 0; i < NITEMS; i++)
        Queue_Enqueue(&queue, (int)(p * NITEMS + i));
    return NULL;
}

static void *consumer(void *arg) {
    (void)arg;
    int last[NPRODUCERS];
    for (int p = 0; p < NPRODUCERS; p++)
        last[p] = -1;

    while (__atomic_load_n(&consumed, __ATOMIC_RELAXED) < TOTAL) {
        int v;
        if (Queue_Dequeue(&queue, &v) != 0) {
            sched_yield();        // queue empty right now; let producers run
            continue;
        }
        __atomic_fetch_add(&seen[v], 1, __ATOMIC_RELAXED);
        __atomic_fetch_add(&consumed, 1, __ATOMIC_RELAXED);

        int p = v / NITEMS, i = v % NITEMS;
        if (i <= last[p])
            __atomic_fetch_add(&order_errors, 1, __ATOMIC_RELAXED);
        last[p] = i;
    }
    return NULL;
}

int main(void) {
    pthread_t prod[NPRODUCERS], cons[NCONSUMERS];

    Queue_Init(&queue);

    for (long i = 0; i < NCONSUMERS; i++)
        pthread_create(&cons[i], NULL, consumer, NULL);
    for (long i = 0; i < NPRODUCERS; i++)
        pthread_create(&prod[i], NULL, producer, (void *)i);
    for (int i = 0; i < NPRODUCERS; i++)
        pthread_join(prod[i], NULL);
    for (int i = 0; i < NCONSUMERS; i++)
        pthread_join(cons[i], NULL);

    int lost = 0, duplicated = 0;
    for (int v = 0; v < TOTAL; v++) {
        if (seen[v] == 0) lost++;
        if (seen[v] > 1) duplicated++;
    }

    int leftover;
    int empty = Queue_Dequeue(&queue, &leftover) == -1;

    printf("values enqueued   = %d\n", TOTAL);
    printf("values dequeued   = %d\n", consumed);
    printf("lost              = %d\n", lost);
    printf("duplicated        = %d\n", duplicated);
    printf("FIFO violations   = %d\n", order_errors);
    printf("queue empty at end: %s\n", empty ? "yes" : "no");

    int ok = consumed == TOTAL && lost == 0 && duplicated == 0 &&
             order_errors == 0 && empty;
    printf("%s\n", ok ? "PASS" : "FAIL");

    free(queue.head);             // the dummy node
    return ok ? 0 : 1;
}
