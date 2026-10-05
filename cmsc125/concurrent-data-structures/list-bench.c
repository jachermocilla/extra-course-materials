/*
 * Benchmark: single-lock linked list vs hand-over-hand linked list
 * (OSTEP, Chapter 29).
 *
 * The list is pre-filled with `size` keys (0..size-1). Each thread then
 * performs `ops` operations: with probability insert_pct percent an
 * insert of a new key, otherwise a lookup of a random key in 0..2*size-1
 * (roughly half of the lookups miss, so they walk the whole list).
 *
 * Build: gcc -O2 -Wall -pthread list-bench.c -o list-bench
 * Usage: ./list-bench <single|hoh> <nthreads> <size> <ops> <insert_pct>
 * Prints the elapsed time in seconds for the threaded phase only.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>

#define MAXTHREADS 256

/* ======================= single-lock list ======================= */
typedef struct __snode_t {
    int key;
    struct __snode_t *next;
} snode_t;

typedef struct __slist_t {
    snode_t *head;
    pthread_mutex_t lock;
} slist_t;

static void single_init(slist_t *L) {
    L->head = NULL;
    pthread_mutex_init(&L->lock, NULL);
}

static void single_insert(slist_t *L, int key) {
    snode_t *new = malloc(sizeof(snode_t));
    if (new == NULL) { perror("malloc"); return; }
    new->key = key;
    pthread_mutex_lock(&L->lock);
    new->next = L->head;
    L->head = new;
    pthread_mutex_unlock(&L->lock);
}

static int single_lookup(slist_t *L, int key) {
    int rv = -1;
    pthread_mutex_lock(&L->lock);
    snode_t *curr = L->head;
    while (curr) {
        if (curr->key == key) { rv = 0; break; }
        curr = curr->next;
    }
    pthread_mutex_unlock(&L->lock);
    return rv;
}

static long single_count_and_free(slist_t *L) {
    long n = 0;
    snode_t *c = L->head;
    while (c) { snode_t *next = c->next; free(c); c = next; n++; }
    return n;
}

/* ===================== hand-over-hand list ====================== */
typedef struct __hnode_t {
    int key;
    struct __hnode_t *next;
    pthread_mutex_t lock;
} hnode_t;

typedef struct __hlist_t {
    hnode_t *head;
    pthread_mutex_t lock;
} hlist_t;

static void hoh_init(hlist_t *L) {
    L->head = NULL;
    pthread_mutex_init(&L->lock, NULL);
}

static void hoh_insert(hlist_t *L, int key) {
    hnode_t *new = malloc(sizeof(hnode_t));
    if (new == NULL) { perror("malloc"); return; }
    new->key = key;
    pthread_mutex_init(&new->lock, NULL);
    pthread_mutex_lock(&L->lock);
    new->next = L->head;
    L->head = new;
    pthread_mutex_unlock(&L->lock);
}

static int hoh_lookup(hlist_t *L, int key) {
    pthread_mutex_lock(&L->lock);
    hnode_t *curr = L->head;
    if (curr) pthread_mutex_lock(&curr->lock);
    pthread_mutex_unlock(&L->lock);
    while (curr) {
        if (curr->key == key) { pthread_mutex_unlock(&curr->lock); return 0; }
        hnode_t *next = curr->next;
        if (next) pthread_mutex_lock(&next->lock);
        pthread_mutex_unlock(&curr->lock);
        curr = next;
    }
    return -1;
}

static long hoh_count_and_free(hlist_t *L) {
    long n = 0;
    hnode_t *c = L->head;
    while (c) {
        hnode_t *next = c->next;
        pthread_mutex_destroy(&c->lock);
        free(c);
        c = next;
        n++;
    }
    return n;
}

/* ========================= driver ========================= */
static slist_t slist;
static hlist_t hlist;
static int use_hoh = 0;
static int size, ops, insert_pct;
static long inserted[MAXTHREADS];   /* inserts done by each thread */

static void *worker(void *arg) {
    long t = (long)arg;
    unsigned int seed = 12345u + (unsigned int)t * 7919u;
    long my_inserts = 0;
    for (int i = 0; i < ops; i++) {
        if ((int)(rand_r(&seed) % 100) < insert_pct) {
            /* new keys are >= 2*size, so they never collide with lookups of 0..2*size-1 */
            int key = 2 * size + (int)(t * ops + i);
            if (use_hoh) hoh_insert(&hlist, key); else single_insert(&slist, key);
            my_inserts++;
        } else {
            int key = (int)(rand_r(&seed) % (unsigned int)(2 * size));
            if (use_hoh) hoh_lookup(&hlist, key); else single_lookup(&slist, key);
        }
    }
    inserted[t] = my_inserts;
    return NULL;
}

static double now(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec + ts.tv_nsec / 1e9;
}

int main(int argc, char **argv) {
    if (argc != 6) {
        fprintf(stderr, "usage: %s <single|hoh> <nthreads> <size> <ops> <insert_pct>\n", argv[0]);
        return 2;
    }
    use_hoh = strcmp(argv[1], "hoh") == 0;
    int nthreads = atoi(argv[2]);
    size = atoi(argv[3]);
    ops = atoi(argv[4]);
    insert_pct = atoi(argv[5]);
    if (nthreads < 1 || nthreads > MAXTHREADS) {
        fprintf(stderr, "nthreads must be 1..%d\n", MAXTHREADS);
        return 2;
    }

    single_init(&slist);
    hoh_init(&hlist);
    for (int k = 0; k < size; k++) {
        if (use_hoh) hoh_insert(&hlist, k); else single_insert(&slist, k);
    }

    pthread_t threads[MAXTHREADS];
    double start = now();
    for (long i = 0; i < nthreads; i++)
        pthread_create(&threads[i], NULL, worker, (void *)i);
    for (int i = 0; i < nthreads; i++)
        pthread_join(threads[i], NULL);
    double elapsed = now() - start;

    /* sanity check: no insert may be lost */
    long expected = size;
    for (int i = 0; i < nthreads; i++) expected += inserted[i];
    long got = use_hoh ? hoh_count_and_free(&hlist) : single_count_and_free(&slist);
    if (got != expected) {
        fprintf(stderr, "wrong result: %ld nodes, expected %ld\n", got, expected);
        return 1;
    }

    printf("%.6f\n", elapsed);
    return 0;
}
