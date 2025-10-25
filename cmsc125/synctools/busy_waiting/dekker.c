/*
 * Adapted from the osbook 3rd ed by EAAlbacea
 *
 * Compile with: gcc dekker.c -o dekker -pthread
 * Run with: ./dekker
 * 
 * This version follows the boilerplate structure where:
 * - incrits[process]: Indicates process wants to enter critical section
 * - turn: Resolves conflicts when both processes want access
 * - other = 1 - process: Calculates the other process ID
 * 
 * The algorithm ensures mutual exclusion through busy waiting.
 */

#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <stdbool.h>

#define TRUE 1
#define FALSE 0

// Shared variables for Dekker's algorithm
volatile bool incrits[2] = {FALSE, FALSE};  // Interest flags
volatile int turn = 0;                       // Whose turn it is

// Shared counter for demonstration
int shared_counter = 0;

// Generic resource usage function based on boilerplate
void use_resource(int process) {
    int other;
    int iterations = 5;  // Limit iterations for demo
    
    for (int i = 0; i < iterations; i++) {
        other = 1 - process;
        incrits[process] = TRUE;
        
        while (incrits[other]) {
            if (turn == other) {
                incrits[process] = FALSE;
                while (turn == other)
                    ; // busy wait
                incrits[process] = TRUE;
            }
        }
        
        /* === CRITICAL SECTION: use resource === */
        printf("Process %d entering critical section\n", process);
        shared_counter++;
        printf("Process %d: Counter = %d\n", process, shared_counter);
        sleep(1);  // Simulate resource usage
        printf("Process %d leaving critical section\n\n", process);
        /* === END CRITICAL SECTION === */
        
        incrits[process] = FALSE;
        turn = other;
        
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
    
    printf("Starting Dekker's Algorithm Demo (Boilerplate Version)\n");
    printf("======================================================\n\n");
    
    // Create two threads (processes)
    pthread_create(&t0, NULL, thread0_func, NULL);
    pthread_create(&t1, NULL, thread1_func, NULL);
    
    // Wait for both threads to complete
    pthread_join(t0, NULL);
    pthread_join(t1, NULL);
    
    printf("======================================================\n");
    printf("Final counter value: %d\n", shared_counter);
    printf("Expected value: 10\n");
    
    return 0;
}


