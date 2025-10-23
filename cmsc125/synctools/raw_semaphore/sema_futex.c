/*
 * Compilation:
 * gcc -o sema_futex sema_futex.c -pthread
 * ./sema_futex
 *
 * Check the man page of futex
 * This is a raw semaphore implementation using:
 * 1. Atomic operations (__sync_* builtins) for thread-safe counter manipulation
 * 2. Linux futex system calls for efficient sleeping/waking
 *
 * How it works:
 * - sem_raw_wait(): Atomically decrements if count > 0, else sleeps via futex
 * - sem_raw_post(): Atomically increments and wakes one waiting thread
 * - Futex avoids busy-waiting by putting threads to sleep in kernel
 *
 * As a mutex:
 * - Initialize with count = 1
 * - wait() locks (decrements 1 -> 0)
 * - post() unlocks (increments 0 -> 1)
 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <linux/futex.h>
#include <sys/syscall.h>
#include <limits.h>

// Raw semaphore structure
typedef struct {
    int count;  // Semaphore counter
} semaphore_t;

// Atomic compare-and-swap wrapper
static inline int atomic_cas(int* ptr, int old_val, int new_val) {
    return __sync_bool_compare_and_swap(ptr, old_val, new_val);
}

// Atomic decrement and return old value
static inline int atomic_dec_fetch(int* ptr) {
    return __sync_fetch_and_sub(ptr, 1);
}

// Atomic increment
static inline void atomic_inc(int* ptr) {
    __sync_fetch_and_add(ptr, 1);
}

// Futex system call wrapper
static long futex(int* uaddr, int futex_op, int val) {
    return syscall(SYS_futex, uaddr, futex_op, val, NULL, NULL, 0);
}

// Initialize semaphore with initial count
void sem_raw_init(semaphore_t* sem, int initial_value) {
    sem->count = initial_value;
}

// Wait (P operation / down)
void sem_raw_wait(semaphore_t* sem) {
    while (1) {
        int old_count = sem->count;
        
        // If count > 0, try to decrement it
        if (old_count > 0) {
            if (atomic_cas(&sem->count, old_count, old_count - 1)) {
                // Successfully decremented, return
                return;
            }
            // CAS failed, retry
            continue;
        }
        
        // Count is 0, need to wait
        // Use futex to sleep until woken
        futex(&sem->count, FUTEX_WAIT, 0);
    }
}

// Post (V operation / up)
void sem_raw_post(semaphore_t* sem) {
    // Increment the counter
    atomic_inc(&sem->count);
    
    // Wake up one waiting thread
    futex(&sem->count, FUTEX_WAKE, 1);
}

// Destroy semaphore (cleanup)
void sem_raw_destroy(semaphore_t* sem) {
    // Nothing to do for this simple implementation
}

// ========== EXAMPLE USAGE AS MUTEX ==========

semaphore_t mutex;
int shared_counter = 0;

void* thread_func(void* arg) {
    int thread_id = *(int*)arg;
    
    for (int i = 0; i < 5; i++) {
        // Lock
        sem_raw_wait(&mutex);
        
        // Critical section
        int temp = shared_counter;
        printf("Thread %d: counter = %d -> ", thread_id, temp);
        usleep(10000);  // Simulate work
        temp++;
        shared_counter = temp;
        printf("%d\n", shared_counter);
        
        // Unlock
        sem_raw_post(&mutex);
        
        usleep(5000);
    }
    
    return NULL;
}

int main() {
    pthread_t threads[3];
    int thread_ids[3];
    
    // Initialize semaphore as mutex (value = 1)
    sem_raw_init(&mutex, 1);
    
    printf("=== Raw Semaphore Implementation Demo ===\n");
    printf("Initial counter: %d\n\n", shared_counter);
    
    // Create threads
    for (int i = 0; i < 3; i++) {
        thread_ids[i] = i + 1;
        pthread_create(&threads[i], NULL, thread_func, &thread_ids[i]);
    }
    
    // Join threads
    for (int i = 0; i < 3; i++) {
        pthread_join(threads[i], NULL);
    }
    
    printf("\nFinal counter: %d (expected: 15)\n", shared_counter);
    
    sem_raw_destroy(&mutex);
    
    return 0;
}


