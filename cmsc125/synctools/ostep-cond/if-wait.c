/*
 * Parent waiting for child: another poor implementation (OSTEP,
 * Chapter 30). This version is BROKEN on purpose.
 *
 * It has the state variable `done`, but the parent tests it with `if` and
 * then goes to sleep. That leaves a gap between "I saw done == 0" and "I am
 * asleep". If the child runs in that gap, it sets done = 1 and signals while
 * nobody is waiting; the signal is lost and the parent sleeps forever.
 *
 * thr_exit() and thr_join() follow the slide. The slide's pthread_cond_wait(&c)
 * has no mutex argument, but the real function requires one, so thr_join()
 * locks a mutex just to make that call.
 *
 * Only main() and the marked "demo only" lines are extras, so the race can
 * be shown reliably and the program can't hang:
 *   - a pause inside the gap in thr_join() lets the child run there
 *   - alarm() ends the program after 3 seconds and reports the hang
 *
 * Build: gcc -O2 -Wall -pthread if-wait.c -o if-wait
 * Run:   ./if-wait race     (default: child runs in the gap, hangs)
 *        ./if-wait lucky    (parent is asleep first, happens to work)
 */
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <pthread.h>

volatile int done = 0;
pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;   // only needed by cond_wait
pthread_cond_t c = PTHREAD_COND_INITIALIZER;

static int lucky = 0;

void thr_exit() {
    done = 1;
    pthread_cond_signal(&c);
}

void *child(void *arg) {
    (void)arg;
    // demo only: lucky -> run after the parent is asleep; race -> run after the
    // parent has checked done, but while it is still in the gap before wait
    usleep((lucky ? 200 : 50) * 1000);
    printf("child\n");
    thr_exit();
    return NULL;
}

void thr_join() {
    if (done == 0) {
        if (!lucky)
            usleep(200 * 1000);   // demo only: the child runs here, in the gap
        pthread_mutex_lock(&m);
        pthread_cond_wait(&c, &m);
        pthread_mutex_unlock(&m);
    }
}

static void on_alarm(int sig) {
    (void)sig;
    const char msg[] = "parent: STUCK in pthread_cond_wait (the signal was lost)\n";
    if (write(1, msg, sizeof(msg) - 1) < 0)
        _exit(2);
    _exit(1);
}

int main(int argc, char *argv[]) {
    lucky = argc > 1 && strcmp(argv[1], "lucky") == 0;

    setvbuf(stdout, NULL, _IOLBF, 0);   // demo only: print lines immediately, even in a pipe
    signal(SIGALRM, on_alarm);
    alarm(3);                           // demo only: give up after 3 seconds

    printf("parent: begin\n");
    pthread_t p;
    pthread_create(&p, NULL, child, NULL);
    thr_join();
    printf("parent: end\n");
    return 0;
}
