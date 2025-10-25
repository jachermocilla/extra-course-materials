/**
 * gcc -o race_cond race_cond.c -pthread
 */

#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

// Shared counter variable (vulnerable to race condition)
int shared_counter = 0;

// Thread function for JACH
void* jach_thread(void* arg) {
    printf("JACH thread started\n");
    
    for (int i = 0; i < 10; i++) {
        // Race condition: non-atomic increment
        int temp = shared_counter;  // Read
        printf("JACH READ: %d\n", temp);
        
        temp = temp + 1;            // Modify
        usleep(100);  // Small delay to increase chance of race condition
        
        shared_counter = temp;      // Write
        printf("JACH WRITE: %d\n", temp);
    }
    
    printf("JACH thread finished\n");
    return NULL;
}

// Thread function for Patrick
void* patrick_thread(void* arg) {
    printf("Patrick thread started\n");
    
    for (int i = 0; i < 10; i++) {
        // Race condition: non-atomic increment
        int temp = shared_counter;  // Read
        printf("Patrick READ: %d\n", temp);
        
        temp = temp + 1;            // Modify
        usleep(100);  // Small delay to increase chance of race condition
        
        shared_counter = temp;      // Write
        printf("Patrick WRITE: %d\n", temp);
    }
    
    printf("Patrick thread finished\n");
    return NULL;
}

int main() {
    pthread_t jach, patrick;
    
    printf("Initial counter value: %d\n", shared_counter);
    printf("Expected final value: 20\n\n");
    
    // Create JACH thread
    if (pthread_create(&jach, NULL, jach_thread, NULL) != 0) {
        perror("Failed to create JACH thread");
        return 1;
    }
    
    // Create Patrick thread
    if (pthread_create(&patrick, NULL, patrick_thread, NULL) != 0) {
        perror("Failed to create Patrick thread");
        return 1;
    }
    
    // Wait for both threads to complete
    pthread_join(jach, NULL);
    pthread_join(patrick, NULL);
    
    printf("\nFinal counter value: %d\n", shared_counter);
    
    if (shared_counter != 20) {
        printf("RACE CONDITION DETECTED! Lost %d increments\n", 
               20 - shared_counter);
    } else {
        printf("No race condition occurred (try running again)\n");
    }
    
    return 0;
}
