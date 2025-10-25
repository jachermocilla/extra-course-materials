/*
 * Adapted from the osbook 3rd ed by EAAlbacea
 * COMPILATION AND EXECUTION:
 * gcc starvation.c -o starvation -pthread
 * ./starvation_demo

 * KEY STARVATION ISSUE:
 * The line "turn = other" forces STRICT ALTERNATION. This means:
 * - Even if Process 0 wants to use the resource 10 times in a row,
 *  it MUST wait for Process 1 to take its turn each time.
 * - If Process 1 is slow or doesn't need the resource, Process 0
 *  is unnecessarily blocked (STARVED) waiting for Process 1.
 * - This violates the progress condition of critical section solutions.
 *
 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>

#define NUM_ITERATIONS 10

int turn = 0;  // Shared variable controlling access

// Counter to track how many times each process enters critical section
int process0_count = 0;
int process1_count = 0;

void use_resource(int process) {
    int other;
    other = 1 - process;
    
    // Wait until it's this process's turn
    while (turn == other) {
        ; // Busy wait
    }
    
    /* Critical Section - use resource */
    printf("Process %d entering critical section (count: %d)\n", 
           process, process == 0 ? ++process0_count : ++process1_count);
    usleep(100000); // Simulate work (100ms)
    
    // Give turn to the other process
    turn = other;
}

void* process_thread(void* arg) {
    int process_id = *(int*)arg;
    
    for (int i = 0; i < NUM_ITERATIONS; i++) {
        printf("Process %d wants to enter critical section (iteration %d)\n", 
               process_id, i + 1);
        use_resource(process_id);
        
        // Process 0 wants to use resource multiple times in a row
        // but Process 1 only wants it occasionally
        if (process_id == 0) {
            usleep(50000); // Process 0 is fast (50ms between attempts)
        } else {
            usleep(50000000); // Process 1 is slow (500ms between attempts)
        }
    }
    
    return NULL;
}

int main() {
    pthread_t thread0, thread1;
    int id0 = 0, id1 = 1;
    
    printf("=== STARVATION DEMONSTRATION ===\n");
    printf("This code uses strict alternation for mutual exclusion.\n");
    printf("Process 0 is fast and wants frequent access.\n");
    printf("Process 1 is slow and only needs occasional access.\n\n");
    
    // Create two threads
    pthread_create(&thread0, NULL, process_thread, &id0);
    pthread_create(&thread1, NULL, process_thread, &id1);
    
    // Wait for both threads to complete
    pthread_join(thread0, NULL);
    pthread_join(thread1, NULL);
    
    printf("\n=== RESULTS ===\n");
    printf("Process 0 entered critical section: %d times\n", process0_count);
    printf("Process 1 entered critical section: %d times\n", process1_count);
    printf("\n=== STARVATION EXPLANATION ===\n");
    printf("Problem: Even though Process 0 is ready to use the resource\n");
    printf("multiple times, it must wait for Process 1's turn each time.\n");
    printf("Process 0 is STARVED while waiting for the slow Process 1,\n");
    printf("even though Process 1 doesn't need the resource frequently.\n");
    printf("\nThis violates the 'progress' requirement: a process ready\n");
    printf("to enter its critical section should not be indefinitely\n");
    printf("delayed by a process not in or wanting to enter its critical section.\n");
    
    return 0;
}


