/*
 * Producer/consumer, basic version with NO synchronization
 * (OSTEP, Chapter 30: Condition Variables). This version is BROKEN on
 * purpose.
 *
 * A single-slot buffer holds one value at a time. put() requires the
 * buffer to be empty (count == 0) and get() requires it to be full
 * (count == 1). Nothing makes the threads take turns, so one of the two
 * asserts fails: the consumer calls get() before anything was put, or the
 * producer calls put() again before the consumer took the last value.
 *
 * put(), get(), producer() and consumer() are the book's code. Only main()
 * is added, plus alarm() so the program can't run forever if the threads
 * happen to line up for a while.
 *
 * Build: gcc -O2 -Wall -pthread producer-consumer-basic.c -o producer-consumer-basic
 * Run:   ./producer-consumer-basic [loops]     (default: 10 values)
 */
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <unistd.h>
#include <pthread.h>

int buffer;
int count = 0;   // initially, empty

void put(int value) {
    assert(count == 0);
    count = 1;
    buffer = value;
}

int get() {
    assert(count == 1);
    count = 0;
    return buffer;
}

void *producer(void *arg) {
    int i;
    int loops = (int)(long)arg;
    for (i = 0; i < loops; i++) {
        put(i);
    }
    return NULL;
}

void *consumer(void *arg) {
    (void)arg;
    while (1) {
        int tmp = get();
        printf("%d\n", tmp);
    }
    return NULL;
}

int main(int argc, char *argv[]) {
    int loops = argc > 1 ? atoi(argv[1]) : 10;

    alarm(3);   // not in the book: stop after 3 seconds

    pthread_t p, c;
    pthread_create(&p, NULL, producer, (void *)(long)loops);
    pthread_create(&c, NULL, consumer, NULL);
    pthread_join(p, NULL);
    pthread_join(c, NULL);   // the consumer never returns on its own
    return 0;
}
