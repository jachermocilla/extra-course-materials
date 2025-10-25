/*
 * Compilation:
 * gcc -o monitor monitor.c -pthread 
 *
 * Reader-Writer Problem using Monitor:
 * 
 * Problem: Multiple threads need to access shared data
 * - Readers: Only read data (multiple can read simultaneously)
 * - Writers: Modify data (need exclusive access)
 *
 * Solution constraints:
 * - Multiple readers can read at the same time
 * - Only one writer at a time
 * - No reader when a writer is active
 * - Writers have priority (to prevent writer starvation)
 *
 * Monitor encapsulates:
 * - Shared data (shared_data variable)
 * - Reader/writer counts
 * - Condition variables (can_read, can_write)
 * - Synchronization logic in start/end operations
 *
 * This implementation gives writers priority to prevent starvation.
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

// ========== MUTEX AND CONDITION VARIABLE ==========

typedef semaphore_t mutex_t;

typedef struct {
    semaphore_t sem;
    volatile int waiters;
} cond_t;

void mutex_init(mutex_t* mutex) {
    sem_raw_init(mutex, 1);
}

void mutex_lock(mutex_t* mutex) {
    sem_raw_wait(mutex);
}

void mutex_unlock(mutex_t* mutex) {
    sem_raw_post(mutex);
}

void cond_init(cond_t* cond) {
    sem_raw_init(&cond->sem, 0);
    cond->waiters = 0;
}

void cond_wait(cond_t* cond, mutex_t* mutex) {
    atomic_fetch_add((volatile int*)&cond->waiters, 1);
    mutex_unlock(mutex);
    sem_raw_wait(&cond->sem);
    atomic_fetch_add((volatile int*)&cond->waiters, -1);
    mutex_lock(mutex);
}

void cond_signal(cond_t* cond) {
    if (cond->waiters > 0) {
        sem_raw_post(&cond->sem);
    }
}

void cond_broadcast(cond_t* cond) {
    int num_waiters = cond->waiters;
    for (int i = 0; i < num_waiters; i++) {
        sem_raw_post(&cond->sem);
    }
}

// ========== MONITOR IMPLEMENTATION ==========

typedef struct {
    mutex_t mutex;
} monitor_t;

void monitor_init(monitor_t* mon) {
    mutex_init(&mon->mutex);
}

void monitor_enter(monitor_t* mon) {
    mutex_lock(&mon->mutex);
}

void monitor_exit(monitor_t* mon) {
    mutex_unlock(&mon->mutex);
}

void monitor_wait(monitor_t* mon, cond_t* cond) {
    cond_wait(cond, &mon->mutex);
}

void monitor_signal(monitor_t* mon, cond_t* cond) {
    cond_signal(cond);
}

void monitor_broadcast(monitor_t* mon, cond_t* cond) {
    cond_broadcast(cond);
}

// ========== READER-WRITER MONITOR ==========

typedef struct {
    monitor_t monitor;
    cond_t can_read;
    cond_t can_write;
    int readers;
    int writers;
    int waiting_writers;
    int shared_data;
} reader_writer_monitor_t;

void reader_writer_init(reader_writer_monitor_t* rw) {
    monitor_init(&rw->monitor);
    cond_init(&rw->can_read);
    cond_init(&rw->can_write);
    rw->readers = 0;
    rw->writers = 0;
    rw->waiting_writers = 0;
    rw->shared_data = 0;
}

void start_read(reader_writer_monitor_t* rw) {
    monitor_enter(&rw->monitor);
    
    // Wait if there's a writer or waiting writers (writer priority)
    while (rw->writers > 0 || rw->waiting_writers > 0) {
        printf("  [Reader waiting...]\n");
        monitor_wait(&rw->monitor, &rw->can_read);
    }
    
    rw->readers++;
    printf("  [Active readers: %d]\n", rw->readers);
    
    monitor_exit(&rw->monitor);
}

void end_read(reader_writer_monitor_t* rw) {
    monitor_enter(&rw->monitor);
    
    rw->readers--;
    printf("  [Active readers: %d]\n", rw->readers);
    
    // If last reader, signal a waiting writer
    if (rw->readers == 0) {
        monitor_signal(&rw->monitor, &rw->can_write);
    }
    
    monitor_exit(&rw->monitor);
}

void start_write(reader_writer_monitor_t* rw) {
    monitor_enter(&rw->monitor);
    
    rw->waiting_writers++;
    
    // Wait while there are active readers or another writer
    while (rw->readers > 0 || rw->writers > 0) {
        printf("  [Writer waiting... (readers: %d, writers: %d)]\n", 
               rw->readers, rw->writers);
        monitor_wait(&rw->monitor, &rw->can_write);
    }
    
    rw->waiting_writers--;
    rw->writers++;
    
    monitor_exit(&rw->monitor);
}

void end_write(reader_writer_monitor_t* rw) {
    monitor_enter(&rw->monitor);
    
    rw->writers--;
    
    // Writer priority: signal waiting writers first
    if (rw->waiting_writers > 0) {
        monitor_signal(&rw->monitor, &rw->can_write);
    } else {
        // No waiting writers, wake all readers
        monitor_broadcast(&rw->monitor, &rw->can_read);
    }
    
    monitor_exit(&rw->monitor);
}

// ========== TEST PROGRAM ==========

reader_writer_monitor_t rw;

void* reader(void* arg) {
    int id = *(int*)arg;
    
    for (int i = 0; i < 4; i++) {
        start_read(&rw);
        
        // Critical section: reading
        printf("Reader %d: READ value = %d\n", id, rw.shared_data);
        usleep(100000);  // Simulate reading time
        
        end_read(&rw);
        
        usleep(200000);  // Time between reads
    }
    
    return NULL;
}

void* writer(void* arg) {
    int id = *(int*)arg;
    
    for (int i = 0; i < 3; i++) {
        start_write(&rw);
        
        // Critical section: writing
        rw.shared_data = id * 100 + i;
        printf("Writer %d: WROTE value = %d\n", id, rw.shared_data);
        usleep(150000);  // Simulate writing time
        
        end_write(&rw);
        
        usleep(250000);  // Time between writes
    }
    
    return NULL;
}

int main() {
    pthread_t readers[4], writers[2];
    int reader_ids[4] = {1, 2, 3, 4};
    int writer_ids[2] = {1, 2};
    
    printf("=== Reader-Writer Monitor Implementation ===\n");
    printf("Writers have priority over readers\n");
    printf("Multiple readers can read simultaneously\n\n");
    
    // Initialize reader-writer monitor
    reader_writer_init(&rw);
    
    // Create reader threads
    for (int i = 0; i < 4; i++) {
        pthread_create(&readers[i], NULL, reader, &reader_ids[i]);
    }
    
    // Create writer threads
    for (int i = 0; i < 2; i++) {
        pthread_create(&writers[i], NULL, writer, &writer_ids[i]);
    }
    
    // Wait for all threads to complete
    for (int i = 0; i < 4; i++) {
        pthread_join(readers[i], NULL);
    }
    for (int i = 0; i < 2; i++) {
        pthread_join(writers[i], NULL);
    }
    
    printf("\n=== Final shared data value: %d ===\n", rw.shared_data);
    printf("=== All threads completed ===\n");
    
    return 0;
}


