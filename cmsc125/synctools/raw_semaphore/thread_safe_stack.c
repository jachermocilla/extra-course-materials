/*
 * Compilation:
 * gcc -o thread_safe_stack thread_safe_stack.c -pthread 
 *
 * Thread-Safe Stack Features:
 * - Dynamic linked-list based implementation
 * - All operations are thread-safe using mutex
 * - Push: O(1) - adds to top
 * - Pop: O(1) - removes from top, returns false if empty
 * - Peek: O(1) - views top without removing
 * - Size: O(1) - returns current size
 * - Destroy: O(n) - frees all memory
 *
 * Operations:
 * - stack_push(): Thread-safe insertion
 * - stack_pop(): Thread-safe removal (returns bool for success)
 * - stack_peek(): Thread-safe viewing without modification
 * - stack_size(): Thread-safe size query
 * - stack_is_empty(): Thread-safe empty check
 *
 * Demo shows:
 * - 2 pusher threads adding items
 * - 2 popper threads removing items
 * - 1 peeker thread viewing items
 * - All operating concurrently without race conditions
 */

#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sched.h>
#include <stdbool.h>

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

// ========== MUTEX WRAPPER ==========

typedef semaphore_t mutex_t;

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

// ========== THREAD-SAFE STACK IMPLEMENTATION ==========

typedef struct stack_node {
    int data;
    struct stack_node* next;
} stack_node_t;

typedef struct {
    stack_node_t* top;
    int size;
    mutex_t mutex;
} thread_safe_stack_t;

// Initialize the stack
void stack_init(thread_safe_stack_t* stack) {
    stack->top = NULL;
    stack->size = 0;
    mutex_init(&stack->mutex);
}

// Push an item onto the stack
void stack_push(thread_safe_stack_t* stack, int value) {
    // Allocate new node
    stack_node_t* node = (stack_node_t*)malloc(sizeof(stack_node_t));
    if (node == NULL) {
        fprintf(stderr, "Memory allocation failed\n");
        return;
    }
    
    node->data = value;
    
    // Lock the stack
    mutex_lock(&stack->mutex);
    
    // Insert at the top
    node->next = stack->top;
    stack->top = node;
    stack->size++;
    
    // Unlock the stack
    mutex_unlock(&stack->mutex);
}

// Pop an item from the stack
bool stack_pop(thread_safe_stack_t* stack, int* value) {
    mutex_lock(&stack->mutex);
    
    // Check if stack is empty
    if (stack->top == NULL) {
        mutex_unlock(&stack->mutex);
        return false;
    }
    
    // Remove top element
    stack_node_t* node = stack->top;
    *value = node->data;
    stack->top = node->next;
    stack->size--;
    
    mutex_unlock(&stack->mutex);
    
    // Free the node
    free(node);
    
    return true;
}

// Peek at the top element without removing it
bool stack_peek(thread_safe_stack_t* stack, int* value) {
    mutex_lock(&stack->mutex);
    
    if (stack->top == NULL) {
        mutex_unlock(&stack->mutex);
        return false;
    }
    
    *value = stack->top->data;
    
    mutex_unlock(&stack->mutex);
    
    return true;
}

// Get the current size of the stack
int stack_size(thread_safe_stack_t* stack) {
    mutex_lock(&stack->mutex);
    int size = stack->size;
    mutex_unlock(&stack->mutex);
    return size;
}

// Check if the stack is empty
bool stack_is_empty(thread_safe_stack_t* stack) {
    mutex_lock(&stack->mutex);
    bool empty = (stack->top == NULL);
    mutex_unlock(&stack->mutex);
    return empty;
}

// Destroy the stack and free all memory
void stack_destroy(thread_safe_stack_t* stack) {
    mutex_lock(&stack->mutex);
    
    // Free all nodes
    stack_node_t* current = stack->top;
    while (current != NULL) {
        stack_node_t* next = current->next;
        free(current);
        current = next;
    }
    
    stack->top = NULL;
    stack->size = 0;
    
    mutex_unlock(&stack->mutex);
    mutex_destroy(&stack->mutex);
}

// ========== TEST/EXAMPLE PROGRAM ==========

thread_safe_stack_t shared_stack;

void* pusher_thread(void* arg) {
    int thread_id = *(int*)arg;
    
    for (int i = 0; i < 10; i++) {
        int value = thread_id * 100 + i;
        stack_push(&shared_stack, value);
        printf("Thread %d: PUSHED %d (size: %d)\n", 
               thread_id, value, stack_size(&shared_stack));
        usleep(50000);  // 50ms
    }
    
    return NULL;
}

void* popper_thread(void* arg) {
    int thread_id = *(int*)arg;
    
    for (int i = 0; i < 10; i++) {
        int value;
        if (stack_pop(&shared_stack, &value)) {
            printf("Thread %d: POPPED %d (size: %d)\n", 
                   thread_id, value, stack_size(&shared_stack));
        } else {
            printf("Thread %d: Stack empty, retrying...\n", thread_id);
            i--;  // Retry
        }
        usleep(80000);  // 80ms (slower than pushers)
    }
    
    return NULL;
}

void* peek_thread(void* arg) {
    int thread_id = *(int*)arg;
    
    for (int i = 0; i < 5; i++) {
        int value;
        if (stack_peek(&shared_stack, &value)) {
            printf("Thread %d: PEEKED %d (size: %d)\n", 
                   thread_id, value, stack_size(&shared_stack));
        } else {
            printf("Thread %d: Stack empty for peek\n", thread_id);
        }
        usleep(100000);  // 100ms
    }
    
    return NULL;
}

int main() {
    pthread_t pushers[2], poppers[2], peeker;
    int pusher_ids[2] = {1, 2};
    int popper_ids[2] = {3, 4};
    int peeker_id = 5;
    
    printf("=== Thread-Safe Stack Implementation ===\n");
    printf("Using raw semaphore-based mutex\n\n");
    
    // Initialize stack
    stack_init(&shared_stack);
    
    // Create threads
    for (int i = 0; i < 2; i++) {
        pthread_create(&pushers[i], NULL, pusher_thread, &pusher_ids[i]);
        pthread_create(&poppers[i], NULL, popper_thread, &popper_ids[i]);
    }
    pthread_create(&peeker, NULL, peek_thread, &peeker_id);
    
    // Wait for all threads
    for (int i = 0; i < 2; i++) {
        pthread_join(pushers[i], NULL);
        pthread_join(poppers[i], NULL);
    }
    pthread_join(peeker, NULL);
    
    printf("\n=== Final State ===\n");
    printf("Stack size: %d\n", stack_size(&shared_stack));
    printf("Stack is %s\n", stack_is_empty(&shared_stack) ? "empty" : "not empty");
    
    // Pop remaining items
    printf("\nRemaining items in stack:\n");
    int value;
    while (stack_pop(&shared_stack, &value)) {
        printf("  %d\n", value);
    }
    
    // Cleanup
    stack_destroy(&shared_stack);
    
    printf("\n=== Stack destroyed ===\n");
    
    return 0;
}


