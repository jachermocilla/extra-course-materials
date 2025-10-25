/* 
COMPILATION AND EXECUTION:
gcc -pthread deadlock.c -o deadlock
./deadlock

THE DEADLOCK SCENARIO:

Timeline showing how deadlock occurs:
Time | Process 0                    | Process 1
-----|------------------------------|-----------------------------
t1   | incrits[0] = TRUE           |
t2   |                              | incrits[1] = TRUE
t3   | while(incrits[1])... WAIT!  |
t4   |                              | while(incrits[0])... WAIT!
t5   | [blocked forever]            | [blocked forever]

Both processes are now stuck waiting for each other!

WHY IT HAPPENS:
- The operations "set flag" and "check other flag" are not atomic
- If both processes set their flags before checking, both will wait
- Neither can proceed because each is waiting for the other
- This is a classic DEADLOCK situation

*/
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <stdbool.h>

#define MAX_ITERATIONS 100

int incrits[2] = {0, 0};  // Shared flags indicating intent to enter
int iteration_count[2] = {0, 0};
volatile bool deadlock_detected = false;

void use_resource(int process) {
    int attempts = 0;
    
    while (iteration_count[process] < MAX_ITERATIONS && !deadlock_detected) {
        // Indicate intent to enter critical section
        incrits[process] = true;
        
        printf("Process %d: Set incrits[%d] = TRUE (iteration %d)\n", 
               process, process, iteration_count[process] + 1);
        
        // Small delay to increase chance of deadlock race condition
        usleep(10);
        
        // Wait while the other process wants to enter
        attempts = 0;
        while (incrits[1 - process]) {
            attempts++;
            
            // Deadlock detection: if we've been waiting too long
            if (attempts > 100000) {
                printf("\n*** DEADLOCK DETECTED! ***\n");
                printf("Process %d is stuck waiting for Process %d\n", 
                       process, 1 - process);
                printf("Both processes have incrits set to TRUE:\n");
                printf("  incrits[0] = %s\n", incrits[0] ? "TRUE" : "FALSE");
                printf("  incrits[1] = %s\n", incrits[1] ? "TRUE" : "FALSE");
                printf("Both processes are waiting for each other - DEADLOCK!\n");
                deadlock_detected = true;
                return;
            }
            
            // Check every 10000 attempts
            if (attempts % 10000 == 0) {
                printf("Process %d: Still waiting... (attempts: %d)\n", 
                       process, attempts);
            }
        }
        
        /* Critical Section - use resource */
        printf("Process %d: ENTERED critical section\n", process);
        iteration_count[process]++;
        usleep(5000); // Simulate work
        printf("Process %d: EXITING critical section\n", process);
        
        // Exit critical section
        incrits[process] = false;
        
        // Small delay before next iteration
        usleep(1000);
    }
}

void* process_thread(void* arg) {
    int process_id = *(int*)arg;
    use_resource(process_id);
    return NULL;
}

int main() {
    pthread_t thread0, thread1;
    int id0 = 0, id1 = 1;
    
    printf("=== DEADLOCK DEMONSTRATION ===\n");
    printf("This code has a race condition that can cause DEADLOCK.\n");
    printf("Starting two processes...\n\n");
    
    // Create two threads
    pthread_create(&thread0, NULL, process_thread, &id0);
    pthread_create(&thread1, NULL, process_thread, &id1);
    
    // Wait for threads with timeout detection
    sleep(5);  // Give them time to run or deadlock
    
    if (deadlock_detected) {
        printf("\n=== DEADLOCK OCCURRED ===\n");
        printf("The program is stuck in deadlock.\n");
        printf("Terminating threads forcefully...\n");
        pthread_cancel(thread0);
        pthread_cancel(thread1);
    }
    
    pthread_join(thread0, NULL);
    pthread_join(thread1, NULL);
    
    printf("\n=== RESULTS ===\n");
    printf("Process 0 completed iterations: %d\n", iteration_count[0]);
    printf("Process 1 completed iterations: %d\n", iteration_count[1]);
    
    printf("\n=== DEADLOCK EXPLANATION ===\n");
    printf("THE RACE CONDITION:\n");
    printf("1. Process 0 executes: incrits[0] = TRUE\n");
    printf("2. Process 1 executes: incrits[1] = TRUE  [CONTEXT SWITCH HERE]\n");
    printf("3. Process 0 checks: while(incrits[1]) -> TRUE, so it waits\n");
    printf("4. Process 1 checks: while(incrits[0]) -> TRUE, so it waits\n");
    printf("5. DEADLOCK! Both are waiting for the other to set flag to FALSE\n");
    printf("\nPROBLEM: Setting the flag and checking are NOT ATOMIC.\n");
    printf("Both processes can set their flags before either checks,\n");
    printf("resulting in both waiting for each other indefinitely.\n");
    
    return 0;
}


