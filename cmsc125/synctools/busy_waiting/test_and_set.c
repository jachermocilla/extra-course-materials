/*
 * Adapted from the osbook 3rd ed by EAAlbacea
 *
 * Compile with: gcc test_and_set.c -o test_and_set -pthread
 * Run with: ./test_and_set
 * 
 * Test-and-Set using x86-64 inline assembly:
 * - Uses the XCHG instruction which is inherently atomic on x86
 * - XCHG locks the memory bus automatically
 * - Returns old value and sets lock to FALSE (0) atomically
 * 
 * How it works:
 * 1. test_and_set atomically reads lock and sets it to FALSE
 * 2. Returns TRUE if lock was available (TRUE -> FALSE = acquired)
 * 3. Returns FALSE if lock was taken (FALSE -> FALSE = still locked)
 * 4. Process busy-waits while test_and_set returns FALSE
 * 5. Enters critical section when test_and_set returns TRUE
 * 6. Releases lock by setting canbeused = TRUE
 * 
 * Assembly breakdown:
 * - movl $0, %eax      : Load FALSE (0) into register eax
 * - xchgl %eax, *lock  : Atomically swap eax with memory location
 * - movl %eax, result  : Store the old value that was in memory
 */

#include <stdio.h>
#include <pthread.h>
#include <unistd.h>
#include <stdbool.h>

#define TRUE 1
#define FALSE 0

// Shared lock variable
volatile int canbeused = TRUE;

// Shared counter for demonstration
int shared_counter = 0;

// Test-and-Set implementation using x86-64 inline assembly
// Returns the old value and sets the variable to FALSE (locked)
int test_and_set(volatile int *lock) {
    int old_value = FALSE;
    
    // Using xchg instruction which is atomic
    // xchg atomically exchanges values between register and memory
    __asm__ __volatile__(
        "movl $0, %%eax\n\t"          // Load FALSE (0) into eax
        "xchgl %%eax, %1\n\t"         // Atomic exchange: swap eax with *lock
        "movl %%eax, %0\n\t"          // Store old value in old_value
        : "=r" (old_value)            // Output: old_value
        : "m" (*lock)                 // Input: memory location of lock
        : "%eax", "memory"            // Clobbered: eax register and memory
    );
    
    return old_value;
}

// Alternative using GCC built-in atomic operation
int test_and_set_builtin(volatile int *lock) {
    // __sync_lock_test_and_set atomically sets lock to FALSE and returns old value
    return __sync_lock_test_and_set(lock, FALSE);
}

// Generic resource usage function based on boilerplate
void use_resource(int process) {
    int iterations = 5;  // Limit iterations for demo
    
    for (int i = 0; i < iterations; i++) {
        // Busy wait until we acquire the lock
        while (!test_and_set(&canbeused))
            ; // busy waiting - spin until lock is acquired
        
        /* === CRITICAL SECTION: use resource === */
        printf("Process %d entering critical section\n", process);
        shared_counter++;
        printf("Process %d: Counter = %d\n", process, shared_counter);
        sleep(1);  // Simulate resource usage
        printf("Process %d leaving critical section\n\n", process);
        /* === END CRITICAL SECTION === */
        
        canbeused = TRUE;  // Release the lock
        
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
    
    printf("Starting Test-and-Set Algorithm Demo (Inline Assembly)\n");
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


