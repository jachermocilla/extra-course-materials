/* uthreads.c 
 *
 * This is a really simple cooperative user thread library that uses
 * setjmp/longjmp to save and restore registers.  There's no notion of
 * "blocked" threads (it does busy looping).
 *
 * It's just for fun!
 *
 * Written by: Dan Williams <danlythemanly@gmail.com> 3/2/14
 */

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <setjmp.h>
#include <string.h>

#define STACK_SIZE 4096
#define ERROR(x...) do {                        \
    fprintf(stderr, x);                         \
    return -1;                                  \
    } while (0)

struct uthread {
    int done;
    unsigned char *stack;
    jmp_buf regs;
};

/* we store state for the init thread at the end 
   of the list (but it has no stack) */
#define INIT_THREAD_ID num_threads

static struct uthread *threads;
static int num_threads;
static int num_done;
static int current_thread;

static void (*worker_fcn)(void *);
static void *worker_arg;

int uthread_id(void) {
    return current_thread;
}

/* round robin ftw */
static int find_next_thread() {
    int n = (current_thread + 1) % num_threads;

    if ( num_done == num_threads )
        return INIT_THREAD_ID;

    while ( threads[n].done )
        n = (n + 1) % num_threads;

    return n;
}

void uthread_schedule(void) {
    int next_thread;
    
    next_thread = find_next_thread();
    if ( !setjmp(threads[current_thread].regs) ) {
        /* we are saving */
        current_thread = next_thread;
        longjmp(threads[next_thread].regs, 1);
        /* this should never be reached */
    }

    /* thread is resumed now! */
}


static void get_born() {
    int i;

    /* this is the birth point for all new threads */
    if ( !setjmp(threads[INIT_THREAD_ID].regs) ) {

        for ( i = 0; i < num_threads + 1; i++ )
            memcpy(threads[i].regs, 
                   threads[INIT_THREAD_ID].regs, sizeof(jmp_buf));

        /* switch to first worker */
        current_thread = 0;
        longjmp(threads[current_thread].regs, 1);

        /* this should never be reached */
    }

    /* if we're back at init, all other threads are finished */
    if ( current_thread == INIT_THREAD_ID ) 
        return;

    /* o/w it's a worker: fix stack/base pointer and call fcn */
    asm volatile("movq %0, %%rsp\n\t" 
                 "movq %0, %%rbp" 
                 : : "r" (&threads[current_thread].stack[STACK_SIZE]));

    worker_fcn(worker_arg);

    /* mark the thread as done and tell someone else to run */
    threads[current_thread].done = 1;
    num_done++;
    uthread_schedule();

    /* no one should get here */
}

static void cleanup_stacks(int n){
    int i;

    for ( i = 0; i < n; i++ ) 
        free(threads[i].stack);
}


int uthread_start_threads(int n, void (*fcn)(void *), void *arg) {
    int i;

    worker_fcn = fcn;
    worker_arg = arg;
    num_threads = n;

    threads = malloc(sizeof(struct uthread) * (num_threads + 1));
    if ( !threads )
        ERROR("couldn't allocate threads!");

    memset(threads, 0, sizeof(struct uthread) * (num_threads + 1));

    for ( i = 0; i < num_threads; i++ ) {
        threads[i].stack = malloc(STACK_SIZE);
        if ( !threads[i].stack ) {
            cleanup_stacks(i - 1);
            free(threads);
            ERROR("couldn't allocate stacks!");
        }    
    }

    current_thread = INIT_THREAD_ID;

    get_born();

    cleanup_stacks(num_threads);
    free(threads);
    return 0;
}
