/*
 * Concurrent linked list with hand-over-hand locking (OSTEP, Chapter 29),
 * also called lock coupling, using pthread_mutex.
 *
 * Instead of one lock for the whole list, every node has its own lock.
 * A traversal grabs the next node's lock BEFORE releasing the current
 * node's lock, so it is never "between" two nodes. Other threads can
 * then work on different parts of the list at the same time.
 *
 * Build: gcc -O2 -Wall -pthread hand-over-hand-linked-list.c -o hand-over-hand-linked-list
 */
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>

// basic node structure (one lock per node)
typedef struct __node_t {
    int key;
    struct __node_t *next;
    pthread_mutex_t lock;
} node_t;

// basic list structure (the lock protects the head pointer)
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
    pthread_mutex_init(&new->lock, NULL);

    // just lock critical section: only the head pointer changes. The new
    // node isn't reachable yet, so it needs no lock of its own.
    pthread_mutex_lock(&L->lock);
    new->next = L->head;
    L->head = new;
    pthread_mutex_unlock(&L->lock);
}

int List_Lookup(list_t *L, int key) {
    // Lock the list (head pointer), then the first node, then let go of
    // the list lock.
    pthread_mutex_lock(&L->lock);
    node_t *curr = L->head;
    if (curr)
        pthread_mutex_lock(&curr->lock);
    pthread_mutex_unlock(&L->lock);

    // Hand over hand: lock the next node, then release the current one.
    while (curr) {
        if (curr->key == key) {
            pthread_mutex_unlock(&curr->lock);
            return 0; // success
        }
        node_t *next = curr->next;
        if (next)
            pthread_mutex_lock(&next->lock);
        pthread_mutex_unlock(&curr->lock);
        curr = next;
    }
    return -1; // failure
}

/* ---------------------------------------------------------------------
 * Test: NTHREADS threads each insert NKEYS distinct keys while another
 * NTHREADS threads do lookups at the same time (to exercise the locking;
 * those results are ignored since a key may not be inserted yet). Then
 * the list is checked: it must hold exactly NTHREADS * NKEYS nodes, every
 * inserted key must be found, and keys never inserted must not be found.
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

static void *reader(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++)
        List_Lookup(&list, (int)(t * NKEYS + i));   // may or may not be there yet
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
    pthread_t ins[NTHREADS], rd[NTHREADS], chk[NTHREADS];

    List_Init(&list);

    // Phase 1: concurrent inserts and lookups.
    for (long i = 0; i < NTHREADS; i++) {
        pthread_create(&ins[i], NULL, inserter, (void *)i);
        pthread_create(&rd[i], NULL, reader, (void *)i);
    }
    for (int i = 0; i < NTHREADS; i++) {
        pthread_join(ins[i], NULL);
        pthread_join(rd[i], NULL);
    }

    // Count nodes (single-threaded now, so no lock needed).
    int count = 0;
    for (node_t *n = list.head; n; n = n->next)
        count++;

    // Phase 2: concurrent verification lookups.
    for (long i = 0; i < NTHREADS; i++)
        pthread_create(&chk[i], NULL, checker, (void *)i);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(chk[i], NULL);

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
        pthread_mutex_destroy(&n->lock);
        free(n);
        n = next;
    }
    return ok ? 0 : 1;
}
