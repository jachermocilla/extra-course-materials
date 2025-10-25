/*
 * Adapted from the osbook 3rd ed by EAAlbacea
 * 
 * Compile with: gcc peterson.c -o peterson -pthread
 * Run with: ./peterson
 * 
 * Peterson's Algorithm - Simpler than Dekker's:
 * - interested[process]: Process wants to enter critical section
 * - turn: Indicates which process gave up priority
 * 
 * Key difference from Dekker's:
 * - Each process sets turn to ITSELF (giving priority to other)
 * - Waits only if: turn is still theirs AND other is interested
 * - More elegant and easier to understand than Dekker's
 * 
 * The algorithm ensures mutual exclusion through a combination of
 * interest indication and courteous turn-giving.
 */

#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <stdbool.h>

#define TRUE 1
#define FALSE 0

// Shared variables for Peterson's algorithm
volatile int turn;
volatile int interested[2] = {FALSE, FALSE};

// Shared counter for demonstration
int shared_counter = 0;

// Generic resource usage function based on boilerplate
void use_resource(int process) {
    int other;
    int iterations = 5;  // Limit iterations for demo
    
    for (int i = 0; i < iterations; i++) {
        other = 1 - process;
        interested[process] = TRUE;
        turn = process;
        
        while (turn == process && interested[other])
            ; // busy waiting
        
        /* === CRITICAL SECTION: use resource === */
        printf("Process %d entering critical section\n", process);
        shared_counter++;
        printf("Process %d: Counter = %d\n", process, shared_counter);
        sleep(1);  // Simulate resource usage
        printf("Process %d leaving critical section\n\n", process);
        /* === END CRITICAL SECTION === */
        
        interested[process] = FALSE;
        
        sleep(1);  // Simulate non-critical work
    }
}

// Thread wrapper for process 0
void* thread0_func(void* arg) {
    use_resource(0);
    return NULL;
}

// Thread wrapper for process 1
void* thread1_func(void* arg) {
    use_resource(1);
    return NULL;
}

int main() {
    pthread_t t0, t1;
    
    printf("Starting Peterson's Algorithm Demo\n");
    printf("===================================\n\n");
    
    // Create two threads (processes)
    pthread_create(&t0, NULL, thread0_func, NULL);
    pthread_create(&t1, NULL, thread1_func, NULL);
    
    // Wait for both threads to complete
    pthread_join(t0, NULL);
    pthread_join(t1, NULL);
    
    printf("===================================\n");
    printf("Final counter value: %d\n", shared_counter);
    printf("Expected value: 10\n");
    
    return 0;
}


