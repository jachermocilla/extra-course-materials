/*
 * Parent waiting for child: spin-based approach (OSTEP, Chapter 30:
 * Condition Variables).
 *
 * The parent creates a child thread, then spins on a shared flag until
 * the child sets it. It works, but the parent burns CPU the whole time
 * it waits. The chapter's fix is a condition variable.
 *
 * Build: gcc -O2 -Wall -pthread spin-based.c -o spin-based
 * Run:   ./spin-based
 */
#include <stdio.h>
#include <pthread.h>

volatile int done = 0;

void *child(void *arg) {
    (void)arg;
    printf("child\n");
    done = 1;
    return NULL;
}

int main(int argc, char *argv[]) {
    (void)argc; (void)argv;
    printf("parent: begin\n");
    pthread_t c;
    pthread_create(&c, NULL, child, NULL); // create child
    while (done == 0)
        ; // spin
    printf("parent: end\n");
    return 0;
}
