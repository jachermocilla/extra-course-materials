#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sched.h>
#include <sys/types.h>
#include <sys/time.h>
#include <string.h>
#include <errno.h>

void busy_work(int duration_ms) {
    struct timeval start, current;
    gettimeofday(&start, NULL);
    
    // Simulate CPU-intensive work
    volatile long counter = 0;
    do {
        counter++;
        gettimeofday(&current, NULL);
    } while (((current.tv_sec - start.tv_sec) * 1000 + 
              (current.tv_usec - start.tv_usec) / 1000) < duration_ms);
}

void print_scheduling_info() {
    int policy = sched_getscheduler(0);
    struct sched_param param;
    sched_getparam(0, &param);
    
    const char* policy_name;
    switch(policy) {
        case SCHED_OTHER: policy_name = "SCHED_OTHER"; break;
        case SCHED_FIFO:  policy_name = "SCHED_FIFO"; break;
        case SCHED_RR:    policy_name = "SCHED_RR"; break;
        default:          policy_name = "UNKNOWN"; break;
    }
    
    printf("PID: %d, Policy: %s, Priority: %d\n", 
           getpid(), policy_name, param.sched_priority);
}

int main(int argc, char *argv[]) {
    if (argc != 3) {
        printf("Usage: %s <priority> <work_duration_ms>\n", argv[0]);
        printf("Priority: 1-99 for SCHED_FIFO, 0 for SCHED_OTHER\n");
        printf("Work duration: milliseconds of CPU work\n");
        return 1;
    }
    
    int priority = atoi(argv[1]);
    int work_duration = atoi(argv[2]);
    
    printf("Starting process with requested priority %d\n", priority);
    
    if (priority > 0) {
        // Set SCHED_FIFO policy
        struct sched_param param;
        param.sched_priority = priority;
        
        if (sched_setscheduler(0, SCHED_FIFO, &param) == -1) {
            printf("Failed to set SCHED_FIFO: %s\n", strerror(errno));
            printf("Note: You may need to run as root or with CAP_SYS_NICE capability\n");
            return 1;
        }
        printf("Successfully set SCHED_FIFO policy\n");
    } else {
        printf("Using default SCHED_OTHER policy\n");
    }
    
    print_scheduling_info();
    
    struct timeval start, end;
    gettimeofday(&start, NULL);
    
    printf("Starting CPU-intensive work for %d ms...\n", work_duration);
    busy_work(work_duration);
    
    gettimeofday(&end, NULL);
    long actual_time = (end.tv_sec - start.tv_sec) * 1000 + 
                      (end.tv_usec - start.tv_usec) / 1000;
    
    printf("Process %d completed work in %ld ms\n", getpid(),actual_time);
    printf("Process finished\n");
    
    return 0;
}
