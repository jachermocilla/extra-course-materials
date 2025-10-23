/*
 * Compilation:
 * gcc -o sema_no_futex sema_no_futex.c -pthread
 *
 * This is a pure spin-lock based semaphore using only:
 * 1. Atomic compare-and-swap operations
 * 2. Busy-waiting with exponential backoff
 * 3. CPU-specific pause/yield instructions
 * 4. sched_yield() for cooperative scheduling
 *
 * How it works:
 * - sem_raw_wait(): Spins trying to atomically decrement count if > 0
 *   Uses exponential backoff to reduce CPU contention
 * - sem_raw_post(): Atomically increments count
 * - No kernel involvement, pure userspace implementation
 *
 * As a mutex:
 * - Initialize with count = 1 (binary semaphore)
 * - wait() acquires lock (1 -> 0)
 * - post() releases lock (0 -> 1)
 *
 * Trade-offs vs futex:
 * + No system calls - faster when contention is low
 * - Burns CPU cycles when waiting - inefficient under high contention
 * - No fair scheduling - threads may starve
 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sched.h>

// Raw semaphore structure
typedef struct {
    volatile int count;  // Semaphore counter (volatile for visibility)
} semaphore_t;

// Atomic compare-and-swap
static inline int atomic_cas(volatile int* ptr, int old_val, int new_val) {
    return __sync_bool_compare_and_swap(ptr, old_val, new_val);
}

// Atomic fetch and add
static inline int atomic_fetch_add(volatile int* ptr, int val) {
    return __sync_fetch_and_add(ptr, val);
}

// CPU pause instruction to reduce contention
static inline void cpu_relax(void) {
#if defined(__x86_64__) || defined(__i386__)
    __asm__ __volatile__("pause" ::: "memory");
#elif defined(__aarch64__)
    __asm__ __volatile__("yield" ::: "memory");
#else
    __asm__ __volatile__("" ::: "memory");
#endif
}

// Initialize semaphore with initial count
void sem_raw_init(semaphore_t* sem, int initial_value) {
    sem->count = initial_value;
}

// Wait (P operation / down) - busy wait with exponential backoff
void sem_raw_wait(semaphore_t* sem) {
    int backoff = 1;
    
    while (1) {
        int current = sem->count;
        
        // If count > 0, try to decrement it atomically
        if (current > 0) {
            if (atomic_cas(&sem->count, current, current - 1)) {
                // Successfully acquired semaphore
                return;
            }
        }
        
        // Exponential backoff to reduce CPU contention
        for (int i = 0; i < backoff; i++) {
            cpu_relax();
        }
        
        // Increase backoff, but cap it
        if (backoff < 1024) {
            backoff *= 2;
        }
        
        // Occasionally yield to other threads
        if (backoff >= 128) {
            sched_yield();
        }
    }
}

// Try wait (non-blocking version)
int sem_raw_trywait(semaphore_t* sem) {
    int current = sem->count;
    
    if (current > 0) {
        if (atomic_cas(&sem->count, current, current - 1)) {
            return 1;  // Success
        }
    }
    return 0;  // Failed to acquire
}

// Post (V operation / up)
void sem_raw_post(semaphore_t* sem) {
    // Atomically increment the counter
    atomic_fetch_add(&sem->count, 1);
}

// Destroy semaphore
void sem_raw_destroy(semaphore_t* sem) {
    // Nothing to do for this implementation
}

// Get current semaphore value (non-standard, for debugging)
int sem_raw_getvalue(semaphore_t* sem) {
    return sem->count;
}

// ========== EXAMPLE USAGE AS MUTEX ==========

semaphore_t mutex;
int shared_counter = 0;

void* thread_func(void* arg) {
    int thread_id = *(int*)arg;
    
    for (int i = 0; i < 5; i++) {
        // Lock (wait on semaphore)
        sem_raw_wait(&mutex);
        
        // Critical section
        int temp = shared_counter;
        printf("Thread %d: counter = %d -> ", thread_id, temp);
        fflush(stdout);
        usleep(10000);  // Simulate work
        temp++;
        shared_counter = temp;
        printf("%d\n", shared_counter);
        
        // Unlock (post to semaphore)
        sem_raw_post(&mutex);
        
        usleep(5000);
    }
    
    return NULL;
}

int main() {
    pthread_t threads[3];
    int thread_ids[3];
    
    // Initialize semaphore as mutex (initial value = 1)
    sem_raw_init(&mutex, 1);
    
    printf("=== Raw Semaphore (No Futex) Implementation ===\n");
    printf("Using atomic operations and spin-wait with backoff\n");
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


