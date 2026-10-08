/*
 * Producer/consumer: the final solution (OSTEP, Chapter 30: Condition
 * Variables).
 *
 * Same locking as two-cv-while.c (two condition variables, `while` loops),
 * but the single slot becomes a circular buffer of MAX slots. The producer
 * only waits when the buffer is full (count == MAX) and the consumer only
 * waits when it is empty (count == 0), so producers and consumers can run
 * concurrently most of the time, with fewer context switches.
 *
 * Note: the slides name the buffer indexes `fill` and `use`, but `fill` is
 * also the name of the condition variable. Both can't exist in one C file,
 * so the indexes are called fill_ptr and use_ptr here.
 *
 * put(), get(), producer() and consumer() follow the slides. Only main() is
 * added (to pick the number of producers and consumers and check the
 * result), plus a timeout so a stuck run can't hang.
 *
 * Build: gcc -O2 -Wall -pthread final-solution.c -o final-solution
 *        (use -DMAX=n to change the buffer size; default 4)
 * Run:   ./final-solution [consumers] [loops] [producers]
 *        (default: 1 consumer, 10 values per producer, 1 producer)
 *        ./final-solution 4 100000 3 > /dev/null    (3 producers, 4 consumers)
 */
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <pthread.h>

#ifndef MAX
#define MAX 4
#endif

int buffer[MAX];
int fill_ptr = 0;
int use_ptr = 0;
int count = 0;

void put(int value) {
    buffer[fill_ptr] = value;
    fill_ptr = (fill_ptr + 1) % MAX;
    count++;
}

int get() {
    int tmp = buffer[use_ptr];
    use_ptr = (use_ptr + 1) % MAX;
    count--;
    return tmp;
}

int loops;
pthread_cond_t empty = PTHREAD_COND_INITIALIZER;
pthread_cond_t fill = PTHREAD_COND_INITIALIZER;
pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;

void *producer(void *arg) {
    (void)arg;
    int i;
    for (i = 0; i < loops; i++) {
        pthread_mutex_lock(&mutex);               // p1
        while (count == MAX)                      // p2
            pthread_cond_wait(&empty, &mutex);    // p3
        put(i);                                   // p4
        pthread_cond_signal(&fill);               // p5
        pthread_mutex_unlock(&mutex);             // p6
    }
    return NULL;
}

void *consumer(void *arg) {
    int i;
    int my_loops = (int)(long)arg;   // this consumer's share of the values
    for (i = 0; i < my_loops; i++) {
        pthread_mutex_lock(&mutex);               // c1
        while (count == 0)                        // c2
            pthread_cond_wait(&fill, &mutex);     // c3
        int tmp = get();                          // c4
        pthread_cond_signal(&empty);              // c5
        pthread_mutex_unlock(&mutex);             // c6
        printf("%d\n", tmp);
    }
    return NULL;
}

static void on_alarm(int sig) {
    (void)sig;
    const char msg[] = "TIMEOUT: the program did not finish in 20 seconds\n";
    if (write(2, msg, sizeof(msg) - 1) < 0)
        _exit(2);
    _exit(1);
}

int main(int argc, char *argv[]) {
    int nconsumers = argc > 1 ? atoi(argv[1]) : 1;
    loops = argc > 2 ? atoi(argv[2]) : 10;
    int nproducers = argc > 3 ? atoi(argv[3]) : 1;
    if (nconsumers < 1 || nconsumers > 64 || nproducers < 1 || nproducers > 64 ||
        loops < 1 || (long)loops * nproducers < nconsumers) {
        fprintf(stderr, "usage: %s [consumers 1..64] [loops] [producers 1..64]\n", argv[0]);
        return 2;
    }

    signal(SIGALRM, on_alarm);
    alarm(20);   // not in the book: give up if the threads get stuck

    // Each producer puts `loops` values, so consumers must take that many in total.
    long total = (long)loops * nproducers;

    pthread_t p[64], c[64];
    for (int i = 0; i < nproducers; i++)
        pthread_create(&p[i], NULL, producer, NULL);
    for (int i = 0; i < nconsumers; i++) {
        // split the values between consumers; the last one takes the remainder
        long share = total / nconsumers;
        if (i == nconsumers - 1)
            share += total % nconsumers;
        pthread_create(&c[i], NULL, consumer, (void *)share);
    }
    for (int i = 0; i < nproducers; i++)
        pthread_join(p[i], NULL);
    for (int i = 0; i < nconsumers; i++)
        pthread_join(c[i], NULL);
    return 0;
}
