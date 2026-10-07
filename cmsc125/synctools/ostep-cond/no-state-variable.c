/*
 * Parent waiting for child WITHOUT the state variable `done` (OSTEP,
 * Chapter 30). This version is BROKEN on purpose.
 *
 * thr_exit() signals and thr_join() waits, but nothing records that the
 * child already finished. If the child runs first, its signal is sent
 * while no thread is asleep on the condition, so it is lost. The parent
 * then calls wait and sleeps forever.
 *
 * thr_exit() and thr_join() are exactly the book's code. Only main() has
 * extras so the bug can be shown reliably and the program can't hang:
 *   - an optional argument chooses which thread runs first
 *   - alarm() ends the program after 3 seconds and reports the hang
 *
 * Build: gcc -O2 -Wall -pthread no-state-variable.c -o no-state-variable
 * Run:   ./no-state-variable child-first     (default: hangs)
 *        ./no-state-variable parent-first    (happens to work)
 */
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <pthread.h>

pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;
pthread_cond_t c = PTHREAD_COND_INITIALIZER;

static int parent_first = 0;

void thr_exit() {
    pthread_mutex_lock(&m);
    pthread_cond_signal(&c);
    pthread_mutex_unlock(&m);
}

void *child(void *arg) {
    (void)arg;
    if (parent_first)
        usleep(200 * 1000);   // demo only: let the parent reach wait first
    printf("child\n");
    thr_exit();
    return NULL;
}

void thr_join() {
    pthread_mutex_lock(&m);
    pthread_cond_wait(&c, &m);
    pthread_mutex_unlock(&m);
}

static void on_alarm(int sig) {
    (void)sig;
    const char msg[] = "parent: STUCK in pthread_cond_wait (the signal was lost)\n";
    if (write(1, msg, sizeof(msg) - 1) < 0)
        _exit(2);
    _exit(1);
}

int main(int argc, char *argv[]) {
    parent_first = argc > 1 && strcmp(argv[1], "parent-first") == 0;

    setvbuf(stdout, NULL, _IOLBF, 0);   // demo only: print lines immediately, even in a pipe
    signal(SIGALRM, on_alarm);
    alarm(3);                 // demo only: give up after 3 seconds

    printf("parent: begin\n");
    pthread_t p;
    pthread_create(&p, NULL, child, NULL);
    if (!parent_first)
        usleep(200 * 1000);   // demo only: let the child run and signal first
    thr_join();
    printf("parent: end\n");
    return 0;
}
