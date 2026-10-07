/*
 * Concurrent linked list with a single lock (OSTEP, Chapter 29),
 * using pthread_mutex. Rewrite: the lock is held only around the
 * critical section (the actual list update), not around malloc.
 *
 * Build: gcc -O2 -Wall -pthread concurrent-linked-list-rewrite.c -o concurrent-linked-list-rewrite
 */
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>

// basic node structure
typedef struct __node_t {
    int key;
    struct __node_t *next;
} node_t;

// basic list structure (one used per list)
typedef struct __list_t {
    node_t *head;
    pthread_mutex_t lock;
} list_t;

void List_Init(list_t *L) {
    L->head = NULL;
    pthread_mutex_init(&L->lock, NULL);
}

void List_Insert(list_t *L, int key) {
    // synchronization not needed
    node_t *new = malloc(sizeof(node_t));
    if (new == NULL) {
        perror("malloc");
        return;
    }
    new->key = key;

    // just lock critical section
    pthread_mutex_lock(&L->lock);
    new->next = L->head;
    L->head = new;
    pthread_mutex_unlock(&L->lock);
}

int List_Lookup(list_t *L, int key) {
    int rv = -1;
    pthread_mutex_lock(&L->lock);
    node_t *curr = L->head;
    while (curr) {
        if (curr->key == key) {
            rv = 0;
            break;
        }
        curr = curr->next;
    }
    pthread_mutex_unlock(&L->lock);
    return rv; // now both success and failure
}

/* ---------------------------------------------------------------------
 * Test: NTHREADS threads each insert NKEYS distinct keys concurrently.
 * Then the list is checked: it must hold exactly NTHREADS * NKEYS nodes,
 * every inserted key must be found (looked up by concurrent threads),
 * and keys that were never inserted must not be found.
 * ------------------------------------------------------------------- */
#define NTHREADS 8
#define NKEYS    1000

static list_t list;
static int lookup_errors[NTHREADS];

static void *inserter(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++)
        List_Insert(&list, (int)(t * NKEYS + i));   // keys are unique per thread
    return NULL;
}

static void *checker(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++) {
        if (List_Lookup(&list, (int)(t * NKEYS + i)) != 0)
            lookup_errors[t]++;                     // inserted key not found
        if (List_Lookup(&list, (int)(-1 - (t * NKEYS + i))) == 0)
            lookup_errors[t]++;                     // never-inserted key found
    }
    return NULL;
}

int main(void) {
    pthread_t threads[NTHREADS];

    List_Init(&list);

    for (long i = 0; i < NTHREADS; i++)
        pthread_create(&threads[i], NULL, inserter, (void *)i);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(threads[i], NULL);

    // Count nodes (single-threaded now, so no lock needed).
    int count = 0;
    for (node_t *n = list.head; n; n = n->next)
        count++;

    for (long i = 0; i < NTHREADS; i++)
        pthread_create(&threads[i], NULL, checker, (void *)i);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(threads[i], NULL);

    int errors = 0;
    for (int i = 0; i < NTHREADS; i++)
        errors += lookup_errors[i];

    int expected = NTHREADS * NKEYS;
    printf("nodes in list = %d\n", count);
    printf("expected      = %d\n", expected);
    printf("lookup errors = %d\n", errors);

    int ok = count == expected && errors == 0;
    printf("%s\n", ok ? "PASS" : "FAIL");

    // Clean up.
    node_t *n = list.head;
    while (n) {
        node_t *next = n->next;
        free(n);
        n = next;
    }
    return ok ? 0 : 1;
}
