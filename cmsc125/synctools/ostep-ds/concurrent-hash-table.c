/*
 * Concurrent hash table (OSTEP, Chapter 29), using pthread_mutex.
 *
 * The table is an array of BUCKETS concurrent linked lists, each with its
 * own lock. A key hashes to one bucket, so operations on different buckets
 * never contend. There is no table-wide lock.
 *
 * Build: gcc -O2 -Wall -pthread concurrent-hash-table.c -o concurrent-hash-table
 */
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>

/* ---------------------------------------------------------------------
 * Concurrent linked list (one lock per list), locking only the
 * critical section in List_Insert.
 * ------------------------------------------------------------------- */
typedef struct __node_t {
    int key;
    struct __node_t *next;
} node_t;

typedef struct __list_t {
    node_t *head;
    pthread_mutex_t lock;
} list_t;

void List_Init(list_t *L) {
    L->head = NULL;
    pthread_mutex_init(&L->lock, NULL);
}

int List_Insert(list_t *L, int key) {
    // synchronization not needed
    node_t *new = malloc(sizeof(node_t));
    if (new == NULL) {
        perror("malloc");
        return -1; // fail
    }
    new->key = key;

    // just lock critical section
    pthread_mutex_lock(&L->lock);
    new->next = L->head;
    L->head = new;
    pthread_mutex_unlock(&L->lock);
    return 0; // success
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
    return rv; // 0 = success, -1 = failure
}

/* ---------------------------------------------------------------------
 * Hash table from the book
 * ------------------------------------------------------------------- */
#define BUCKETS (101)

typedef struct __hash_t {
    list_t lists[BUCKETS];
} hash_t;

void Hash_Init(hash_t *H) {
    int i;
    for (i = 0; i < BUCKETS; i++) {
        List_Init(&H->lists[i]);
    }
}

int Hash_Insert(hash_t *H, int key) {
    int bucket = key % BUCKETS;
    return List_Insert(&H->lists[bucket], key);
}

int Hash_Lookup(hash_t *H, int key) {
    int bucket = key % BUCKETS;
    return List_Lookup(&H->lists[bucket], key);
}

/* ---------------------------------------------------------------------
 * Test: NTHREADS threads each insert NKEYS distinct keys while another
 * NTHREADS threads do lookups at the same time (results ignored, since a
 * key may not be inserted yet). Then the table is checked: it must hold
 * exactly NTHREADS * NKEYS nodes, every inserted key must be found, and
 * keys that were never inserted must not be found.
 * (Keys are non-negative: a negative key would give a negative bucket.)
 * ------------------------------------------------------------------- */
#define NTHREADS 8
#define NKEYS    10000
#define ABSENT   (NTHREADS * NKEYS)    // keys >= ABSENT are never inserted

static hash_t table;
static int lookup_errors[NTHREADS];
static int insert_errors[NTHREADS];

static void *inserter(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++)
        if (Hash_Insert(&table, (int)(t * NKEYS + i)) != 0)   // unique per thread
            insert_errors[t]++;
    return NULL;
}

static void *reader(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++)
        Hash_Lookup(&table, (int)(t * NKEYS + i));   // may or may not be there yet
    return NULL;
}

static void *checker(void *arg) {
    long t = (long)arg;
    for (int i = 0; i < NKEYS; i++) {
        if (Hash_Lookup(&table, (int)(t * NKEYS + i)) != 0)
            lookup_errors[t]++;                       // inserted key not found
        if (Hash_Lookup(&table, ABSENT + (int)(t * NKEYS + i)) == 0)
            lookup_errors[t]++;                       // never-inserted key found
    }
    return NULL;
}

int main(void) {
    pthread_t ins[NTHREADS], rd[NTHREADS], chk[NTHREADS];

    Hash_Init(&table);

    // Phase 1: concurrent inserts and lookups.
    for (long i = 0; i < NTHREADS; i++) {
        pthread_create(&ins[i], NULL, inserter, (void *)i);
        pthread_create(&rd[i], NULL, reader, (void *)i);
    }
    for (int i = 0; i < NTHREADS; i++) {
        pthread_join(ins[i], NULL);
        pthread_join(rd[i], NULL);
    }

    // Count nodes and find the bucket sizes (single-threaded now).
    int count = 0, min = -1, max = 0, misplaced = 0;
    for (int b = 0; b < BUCKETS; b++) {
        int n = 0;
        for (node_t *c = table.lists[b].head; c; c = c->next) {
            n++;
            if (c->key % BUCKETS != b)
                misplaced++;
        }
        count += n;
        if (min < 0 || n < min) min = n;
        if (n > max) max = n;
    }

    // Phase 2: concurrent verification lookups.
    for (long i = 0; i < NTHREADS; i++)
        pthread_create(&chk[i], NULL, checker, (void *)i);
    for (int i = 0; i < NTHREADS; i++)
        pthread_join(chk[i], NULL);

    int errors = 0, ierrors = 0;
    for (int i = 0; i < NTHREADS; i++) {
        errors += lookup_errors[i];
        ierrors += insert_errors[i];
    }

    int expected = NTHREADS * NKEYS;
    printf("nodes in table  = %d\n", count);
    printf("expected        = %d\n", expected);
    printf("bucket sizes    = %d to %d (%d buckets)\n", min, max, BUCKETS);
    printf("misplaced keys  = %d\n", misplaced);
    printf("insert failures = %d\n", ierrors);
    printf("lookup errors   = %d\n", errors);

    int ok = count == expected && misplaced == 0 && ierrors == 0 && errors == 0;
    printf("%s\n", ok ? "PASS" : "FAIL");

    // Clean up.
    for (int b = 0; b < BUCKETS; b++) {
        node_t *n = table.lists[b].head;
        while (n) {
            node_t *next = n->next;
            free(n);
            n = next;
        }
    }
    return ok ? 0 : 1;
}
