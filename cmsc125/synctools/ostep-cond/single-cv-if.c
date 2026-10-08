/*
 * Producer/consumer: single condition variable and `if` (OSTEP,
 * Chapter 30: Condition Variables).
 *
 * One condition variable `cond` and one lock `mutex`. The producer waits
 * while the buffer is full and the consumer waits while it is empty, each
 * testing the condition with `if`.
 *
 * With ONE producer and ONE consumer this works. With more than one
 * consumer it is BROKEN: a consumer that is woken up does not re-check the
 * condition, so it can run after another consumer already took the value
 * and call get() on an empty buffer (the assert in get() fails).
 *
 * put(), get(), producer() and consumer() follow the slides. Only main() is
 * added (to pick the number of consumers), plus alarm() so a stuck run
 * can't hang.
 *
 * Build: gcc -O2 -Wall -pthread single-cv-if.c -o single-cv-if
 * Run:   ./single-cv-if [consumers] [loops]    (default: 1 consumer, 10 values)
 *        ./single-cv-if 2 100000 > /dev/null   (likely fails the assert)
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

int loops;
pthread_cond_t cond = PTHREAD_COND_INITIALIZER;
pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;

void *producer(void *arg) {
    (void)arg;
    int i;
    for (i = 0; i < loops; i++) {
        pthread_mutex_lock(&mutex);               // p1
        if (count == 1)                           // p2
            pthread_cond_wait(&cond, &mutex);     // p3
        put(i);                                   // p4
        pthread_cond_signal(&cond);               // p5
        pthread_mutex_unlock(&mutex);             // p6
    }
    return NULL;
}

void *consumer(void *arg) {
    int i;
    int my_loops = (int)(long)arg;   // this consumer's share of the values
    for (i = 0; i < my_loops; i++) {
        pthread_mutex_lock(&mutex);               // c1
        if (count == 0)                           // c2
            pthread_cond_wait(&cond, &mutex);     // c3
        int tmp = get();                          // c4
        pthread_cond_signal(&cond);               // c5
        pthread_mutex_unlock(&mutex);             // c6
        printf("%d\n", tmp);
    }
    return NULL;
}

int main(int argc, char *argv[]) {
    int nconsumers = argc > 1 ? atoi(argv[1]) : 1;
    loops = argc > 2 ? atoi(argv[2]) : 10;
    if (nconsumers < 1 || nconsumers > 64 || loops < nconsumers) {
        fprintf(stderr, "usage: %s [consumers 1..64] [loops >= consumers]\n", argv[0]);
        return 2;
    }

    alarm(20);   // not in the book: give up if the threads get stuck

    pthread_t p, c[64];
    pthread_create(&p, NULL, producer, NULL);
    for (int i = 0; i < nconsumers; i++) {
        // split the values between consumers; the last one takes the remainder
        long share = loops / nconsumers;
        if (i == nconsumers - 1)
            share += loops % nconsumers;
        pthread_create(&c[i], NULL, consumer, (void *)share);
    }
    pthread_join(p, NULL);
    for (int i = 0; i < nconsumers; i++)
        pthread_join(c[i], NULL);
    return 0;
}
