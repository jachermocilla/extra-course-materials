/*
 * Compilation:
 * gcc -o cond_var cond_var.c -pthread 
 *
 * This implements condition variables using raw semaphores:
 *
 * Structure:
 * - cond_t contains a semaphore (for blocking) and waiter count
 * - mutex_t is just a semaphore initialized to 1
 *
 * Operations:
 * - cond_wait(): Releases mutex, blocks on semaphore, re-acquires mutex
 * - cond_signal(): Posts to semaphore if waiters exist (wakes one thread)
 * - cond_broadcast(): Posts multiple times to wake all waiters
 *
 * Example demonstrates classic producer-consumer problem:
 * - Bounded buffer of size 5
 * - Producer waits when buffer is full (cond_wait on not_full)
 * - Consumer waits when buffer is empty (cond_wait on not_empty)
 * - Proper synchronization prevents race conditions
 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sched.h>

// ========== RAW SEMAPHORE IMPLEMENTATION ==========

typedef struct {
    volatile int count;
} semaphore_t;

static inline int atomic_cas(volatile int* ptr, int old_val, int new_val) {
    return __sync_bool_compare_and_swap(ptr, old_val, new_val);
}

static inline int atomic_fetch_add(volatile int* ptr, int val) {
    return __sync_fetch_and_add(ptr, val);
}

static inline void cpu_relax(void) {
#if defined(__x86_64__) || defined(__i386__)
    __asm__ __volatile__("pause" ::: "memory");
#elif defined(__aarch64__)
    __asm__ __volatile__("yield" ::: "memory");
#else
    __asm__ __volatile__("" ::: "memory");
#endif
}

void sem_raw_init(semaphore_t* sem, int initial_value) {
    sem->count = initial_value;
}

void sem_raw_wait(semaphore_t* sem) {
    int backoff = 1;
    while (1) {
        int current = sem->count;
        if (current > 0) {
            if (atomic_cas(&sem->count, current, current - 1)) {
                return;
            }
        }
        for (int i = 0; i < backoff; i++) {
            cpu_relax();
        }
        if (backoff < 1024) {
            backoff *= 2;
        }
        if (backoff >= 128) {
            sched_yield();
        }
    }
}

void sem_raw_post(semaphore_t* sem) {
    atomic_fetch_add(&sem->count, 1);
}

void sem_raw_destroy(semaphore_t* sem) {
    // Nothing to do
}

// ========== CONDITION VARIABLE IMPLEMENTATION ==========

typedef struct {
    semaphore_t sem;           // Semaphore for waiting threads
    volatile int waiters;      // Number of waiting threads
} cond_t;

typedef semaphore_t mutex_t;

// Initialize mutex
void mutex_init(mutex_t* mutex) {
    sem_raw_init(mutex, 1);
}

void mutex_lock(mutex_t* mutex) {
    sem_raw_wait(mutex);
}

void mutex_unlock(mutex_t* mutex) {
    sem_raw_post(mutex);
}

void mutex_destroy(mutex_t* mutex) {
    sem_raw_destroy(mutex);
}

// Initialize condition variable
void cond_init(cond_t* cond) {
    sem_raw_init(&cond->sem, 0);  // Initially no signals
    cond->waiters = 0;
}

// Wait on condition variable
// The mutex must be locked when calling this function
void cond_wait(cond_t* cond, mutex_t* mutex) {
    // Increment waiter count
    atomic_fetch_add((volatile int*)&cond->waiters, 1);
    
    // Release the mutex
    mutex_unlock(mutex);
    
    // Wait on the semaphore (blocks until signaled)
    sem_raw_wait(&cond->sem);
    
    // Decrement waiter count
    atomic_fetch_add((volatile int*)&cond->waiters, -1);
    
    // Re-acquire the mutex
    mutex_lock(mutex);
}

// Signal one waiting thread
void cond_signal(cond_t* cond) {
    // Only signal if there are waiters
    if (cond->waiters > 0) {
        sem_raw_post(&cond->sem);
    }
}

// Broadcast to all waiting threads
void cond_broadcast(cond_t* cond) {
    int num_waiters = cond->waiters;
    
    // Wake up all waiting threads
    for (int i = 0; i < num_waiters; i++) {
        sem_raw_post(&cond->sem);
    }
}

// Destroy condition variable
void cond_destroy(cond_t* cond) {
    sem_raw_destroy(&cond->sem);
}

// ========== PRODUCER-CONSUMER EXAMPLE ==========

#define BUFFER_SIZE 5
#define NUM_ITEMS 20

typedef struct {
    int buffer[BUFFER_SIZE];
    int count;
    int in;
    int out;
    mutex_t mutex;
    cond_t not_empty;
    cond_t not_full;
} bounded_buffer_t;

bounded_buffer_t bb;

void buffer_init(bounded_buffer_t* b) {
    b->count = 0;
    b->in = 0;
    b->out = 0;
    mutex_init(&b->mutex);
    cond_init(&b->not_empty);
    cond_init(&b->not_full);
}

void buffer_put(bounded_buffer_t* b, int item) {
    mutex_lock(&b->mutex);
    
    // Wait while buffer is full
    while (b->count == BUFFER_SIZE) {
        printf("Producer waiting (buffer full)...\n");
        cond_wait(&b->not_full, &b->mutex);
    }
    
    // Add item to buffer
    b->buffer[b->in] = item;
    b->in = (b->in + 1) % BUFFER_SIZE;
    b->count++;
    
    printf("Produced: %d (buffer count: %d)\n", item, b->count);
    
    // Signal that buffer is not empty
    cond_signal(&b->not_empty);
    
    mutex_unlock(&b->mutex);
}

int buffer_get(bounded_buffer_t* b) {
    mutex_lock(&b->mutex);
    
    // Wait while buffer is empty
    while (b->count == 0) {
        printf("Consumer waiting (buffer empty)...\n");
        cond_wait(&b->not_empty, &b->mutex);
    }
    
    // Remove item from buffer
    int item = b->buffer[b->out];
    b->out = (b->out + 1) % BUFFER_SIZE;
    b->count--;
    
    printf("Consumed: %d (buffer count: %d)\n", item, b->count);
    
    // Signal that buffer is not full
    cond_signal(&b->not_full);
    
    mutex_unlock(&b->mutex);
    
    return item;
}

void* producer(void* arg) {
    int id = *(int*)arg;
    
    for (int i = 0; i < NUM_ITEMS / 2; i++) {
        int item = id * 100 + i;
        buffer_put(&bb, item);
        usleep(100000);  // 100ms
    }
    
    return NULL;
}

void* consumer(void* arg) {
    int id = *(int*)arg;
    
    for (int i = 0; i < NUM_ITEMS / 2; i++) {
        int item = buffer_get(&bb);
        usleep(150000);  // 150ms (slower than producer)
    }
    
    return NULL;
}

int main() {
    pthread_t prod_thread, cons_thread;
    int prod_id = 1, cons_id = 1;
    
    printf("=== Condition Variables with Raw Semaphores ===\n");
    printf("Producer-Consumer Problem Demo\n");
    printf("Buffer size: %d\n\n", BUFFER_SIZE);
    
    buffer_init(&bb);
    
    // Create producer and consumer threads
    pthread_create(&prod_thread, NULL, producer, &prod_id);
    pthread_create(&cons_thread, NULL, consumer, &cons_id);
    
    // Wait for completion
    pthread_join(prod_thread, NULL);
    pthread_join(cons_thread, NULL);
    
    printf("\n=== All items produced and consumed ===\n");
    
    // Cleanup
    mutex_destroy(&bb.mutex);
    cond_destroy(&bb.not_empty);
    cond_destroy(&bb.not_full);
    
    return 0;
}


