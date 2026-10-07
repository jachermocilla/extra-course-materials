/*
 * Parent waiting for child: use a condition variable (OSTEP, Chapter 30).
 *
 * The parent sleeps in pthread_cond_wait() until the child signals that
 * it is done. While asleep the parent uses no CPU.
 *
 * Build: gcc -O2 -Wall -pthread condition-variable.c -o condition-variable
 * Run:   ./condition-variable
 */
#include <stdio.h>
#include <pthread.h>

int done = 0;
pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;
pthread_cond_t c = PTHREAD_COND_INITIALIZER;

void thr_exit() {
    pthread_mutex_lock(&m);
    done = 1;
    pthread_cond_signal(&c);
    pthread_mutex_unlock(&m);
}

void *child(void *arg) {
    (void)arg;
    printf("child\n");
    thr_exit();
    return NULL;
}

void thr_join() {
    pthread_mutex_lock(&m);
    while (done == 0)
        pthread_cond_wait(&c, &m);
    pthread_mutex_unlock(&m);
}

int main(int argc, char *argv[]) {
    (void)argc; (void)argv;
    printf("parent: begin\n");
    pthread_t p;
    pthread_create(&p, NULL, child, NULL);
    thr_join();
    printf("parent: end\n");
    return 0;
}
