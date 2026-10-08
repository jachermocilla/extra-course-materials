/*
 * Covering conditions (OSTEP, Chapter 30: Condition Variables).
 *
 * The book's example is a simple memory allocator. allocate(size) waits
 * until enough bytes are free; free() returns bytes and must wake waiting
 * threads. But which thread should it wake? Waiters need different amounts:
 *
 *   signal    wakes ONE waiter. If that waiter needs more bytes than are
 *             free, it goes back to sleep, and a waiter that needed less
 *             (and could now run) stays asleep. Nobody wakes it.
 *   broadcast wakes ALL waiters. Each re-checks `bytesLeft < size` in its
 *             while loop; those that can run do, the rest sleep again.
 *             This is a "covering condition": it covers every case in which
 *             a thread might need to wake, at the cost of waking some
 *             threads for nothing.
 *
 * The book calls the functions allocate() and free(); here they are
 * mem_allocate() and mem_free() so they don't clash with the C library's
 * free(). mem_free() takes the size, as in the book.
 *
 * Only main() is added: a demo that makes the problem happen in a fixed
 * order. Pick the wake-up method on the command line.
 *
 * Build: gcc -O2 -Wall -pthread covering-conditions.c -o covering-conditions
 * Run:   ./covering-conditions broadcast   (works: the covering condition)
 *        ./covering-conditions signal      (wakes the wrong thread, B stays asleep)
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <pthread.h>

#define MAX_HEAP_SIZE 100

// how many bytes of the heap are free?
int bytesLeft = MAX_HEAP_SIZE;

// need lock and condition too
pthread_cond_t c = PTHREAD_COND_INITIALIZER;
pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;

static int use_broadcast = 1;

void *mem_allocate(int size) {
    pthread_mutex_lock(&m);
    while (bytesLeft < size)
        pthread_cond_wait(&c, &m);
    void *ptr = malloc(size);   // get mem from heap
    bytesLeft -= size;
    pthread_mutex_unlock(&m);
    return ptr;
}

void mem_free(void *ptr, int size) {
    pthread_mutex_lock(&m);
    free(ptr);
    bytesLeft += size;
    if (use_broadcast)
        pthread_cond_broadcast(&c);   // covering condition: wake everyone
    else
        pthread_cond_signal(&c);      // whom to signal??
    pthread_mutex_unlock(&m);
}

/* ---------------------------------------------------------------------
 * Demo (not in the book). The heap has 100 bytes.
 *
 *   main takes two chunks: 90 bytes and 10 bytes, so the heap is full.
 *   Thread A asks for 100 bytes and goes to sleep first.
 *   Thread B asks for 10 bytes and goes to sleep second.
 *   main frees the 10-byte chunk. Now 10 bytes are free: B could run, A
 *   could not. Does B wake up?
 * ------------------------------------------------------------------- */
static volatile int a_done = 0, b_got_memory = 0, b_done = 0;

static void *thread_a(void *arg) {
    (void)arg;
    printf("A: asking for 100 bytes\n");
    void *p = mem_allocate(100);
    printf("A: got 100 bytes\n");
    mem_free(p, 100);
    a_done = 1;
    return NULL;
}

static void *thread_b(void *arg) {
    (void)arg;
    printf("B: asking for 10 bytes\n");
    void *p = mem_allocate(10);
    b_got_memory = 1;
    printf("B: got 10 bytes\n");
    mem_free(p, 10);
    b_done = 1;
    return NULL;
}

static void wait_for(volatile int *flag, int ms) {
    for (int i = 0; i < ms && !*flag; i++)
        usleep(1000);
}

int main(int argc, char *argv[]) {
    use_broadcast = !(argc > 1 && strcmp(argv[1], "signal") == 0);
    setvbuf(stdout, NULL, _IOLBF, 0);

    printf("main: wake-up method = %s\n", use_broadcast ? "broadcast" : "signal");

    void *big = mem_allocate(90);
    void *small = mem_allocate(10);
    printf("main: heap is full (bytesLeft = %d)\n", bytesLeft);

    pthread_t a, b;
    pthread_create(&a, NULL, thread_a, NULL);
    usleep(200 * 1000);                 // let A go to sleep first
    pthread_create(&b, NULL, thread_b, NULL);
    usleep(200 * 1000);                 // let B go to sleep second

    printf("main: freeing 10 bytes (A needs 100, B needs 10)\n");
    mem_free(small, 10);

    wait_for(&b_got_memory, 1000);
    if (!b_got_memory) {
        printf("main: B is STILL ASLEEP although 10 bytes are free (bytesLeft = %d).\n", bytesLeft);
        printf("main: the signal woke A, which needs 100, so A went back to sleep.\n");
        return 1;
    }
    wait_for(&b_done, 1000);

    printf("main: freeing 90 bytes\n");
    mem_free(big, 90);
    wait_for(&a_done, 1000);
    pthread_join(a, NULL);
    pthread_join(b, NULL);
    printf("main: all threads finished\n");
    return 0;
}
