/*
 * Benchmark helper: parent waits for a child that works for `delay_ms`
 * milliseconds (it sleeps, so the child itself uses no CPU). The parent
 * waits either by spinning (Ch. 30 spin-based) or on a condition variable.
 * Prints: wall seconds, CPU seconds used by the whole process.
 * Since the child sleeps, the CPU time is the cost of WAITING.
 *
 * Build: gcc -O2 -Wall -pthread wait-bench.c -o wait-bench
 * Usage: ./wait-bench <spin|cond> <delay_ms>
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>
#include <pthread.h>
#include <sys/resource.h>

static volatile int spin_done = 0;

static int done = 0;
static pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t c = PTHREAD_COND_INITIALIZER;

static int use_cond;
static int delay_ms;

static void *child(void *arg) {
    (void)arg;
    usleep((useconds_t)delay_ms * 1000);       /* pretend to work */
    if (use_cond) {
        pthread_mutex_lock(&m);
        done = 1;
        pthread_cond_signal(&c);
        pthread_mutex_unlock(&m);
    } else {
        spin_done = 1;
    }
    return NULL;
}

static double now(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec + ts.tv_nsec / 1e9;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        fprintf(stderr, "usage: %s <spin|cond> <delay_ms>\n", argv[0]);
        return 2;
    }
    use_cond = strcmp(argv[1], "cond") == 0;
    delay_ms = atoi(argv[2]);

    double start = now();
    pthread_t p;
    pthread_create(&p, NULL, child, NULL);
    if (use_cond) {
        pthread_mutex_lock(&m);
        while (done == 0)
            pthread_cond_wait(&c, &m);
        pthread_mutex_unlock(&m);
    } else {
        while (spin_done == 0)
            ;
    }
    double wall = now() - start;
    pthread_join(p, NULL);

    struct rusage ru;
    getrusage(RUSAGE_SELF, &ru);
    double cpu = ru.ru_utime.tv_sec + ru.ru_utime.tv_usec / 1e6 +
                 ru.ru_stime.tv_sec + ru.ru_stime.tv_usec / 1e6;
    printf("%.6f %.6f\n", wall, cpu);
    return 0;
}
