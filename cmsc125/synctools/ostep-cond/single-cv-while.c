/*
 * Producer/consumer: single condition variable and `while` (OSTEP,
 * Chapter 30: Condition Variables).
 *
 * Same as single-cv-if.c, but each thread tests its condition with `while`
 * instead of `if`. A thread that wakes up re-checks the condition and goes
 * back to sleep if it is still false, so a consumer can no longer call get()
 * on an empty buffer. The assert failure from the `if` version is gone.
 *
 * One problem is left: there is a single condition variable, so a signal can
 * wake the wrong kind of thread. A consumer can wake another consumer
 * instead of the producer; that consumer finds the buffer empty and sleeps
 * again, and every thread ends up asleep. The next slides fix this with two
 * condition variables. This shows up as a hang, which alarm() reports.
 *
 * put(), get(), producer() and consumer() follow the slides. Only main() is
 * added (to pick the number of consumers), plus a timeout so a stuck run
 * can't hang.
 *
 * Build: gcc -O2 -Wall -pthread single-cv-while.c -o single-cv-while
 * Run:   ./single-cv-while [consumers] [loops]    (default: 1 consumer, 10 values)
 *        ./single-cv-while 2 100000 > /dev/null   (may hang with 2 or more consumers)
 */
#include <stdio.h>
#include <stdlib.h>
#include <assert.h>
#include <unistd.h>
#include <signal.h>
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
        while (count == 1)                        // p2
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
        while (count == 0)                        // c2
            pthread_cond_wait(&cond, &mutex);     // c3
        int tmp = get();                          // c4
        pthread_cond_signal(&cond);               // c5
        pthread_mutex_unlock(&mutex);             // c6
        printf("%d\n", tmp);
    }
    return NULL;
}

static void on_alarm(int sig) {
    (void)sig;
    const char msg[] = "STUCK: every thread is asleep (the signal woke the wrong thread)\n";
    if (write(2, msg, sizeof(msg) - 1) < 0)
        _exit(2);
    _exit(1);
}

int main(int argc, char *argv[]) {
    int nconsumers = argc > 1 ? atoi(argv[1]) : 1;
    loops = argc > 2 ? atoi(argv[2]) : 10;
    if (nconsumers < 1 || nconsumers > 64 || loops < nconsumers) {
        fprintf(stderr, "usage: %s [consumers 1..64] [loops >= consumers]\n", argv[0]);
        return 2;
    }

    signal(SIGALRM, on_alarm);
    alarm(5);    // not in the book: give up if every thread is asleep

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
