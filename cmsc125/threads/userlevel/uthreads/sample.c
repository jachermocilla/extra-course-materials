#include <stdio.h>
#include <stdlib.h>
#include "uthreads.h"

void do_work(void *ignored){
    int i;

    for ( i = 0; i < 5; i++ ) {
        printf("%d: hello (%d)\n", uthread_id(), i);

        /* flip a coin and decide whether or 
           not to let someone else run */
        if ( rand() < RAND_MAX/2 ) 
            uthread_schedule();
    }
    
    printf("%d: exiting\n", uthread_id());
} 


int main(int argc, char **argv) {
    int num_threads = atoi(argv[1]);

    if ( uthread_start_threads(num_threads, do_work, NULL) )
        printf("we had an error!\n");
    
    printf("all threads are done!\n");
    exit(0);
}

