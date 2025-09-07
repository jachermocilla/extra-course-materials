
#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sched.h>
#include <sys/types.h>
#include <sys/resource.h>
#include <time.h>
#include <string.h>
#include <errno.h>
#include <signal.h>
#include <sys/time.h>

#define SIZE 128
#define MAX_LINE 256

volatile sig_atomic_t keep_running = 1;

void signal_handler(int sig) {
    keep_running = 0;
    printf("\nReceived signal %d, shutting down gracefully...\n", sig);
}

const char* get_policy_name(int policy) {
    switch(policy) {
        case SCHED_OTHER: return "SCHED_OTHER";
        case SCHED_FIFO: return "SCHED_FIFO";
        case SCHED_RR: return "SCHED_RR";
        case SCHED_BATCH: return "SCHED_BATCH";
        //case SCHED_IDLE: return "SCHED_IDLE";
#ifdef SCHED_DEADLINE
        case SCHED_DEADLINE: return "SCHED_DEADLINE";
#endif
        default: return "UNKNOWN";
    }
}

// Function to convert state character to human-readable string
const char* state_to_string(char state) {
    switch (state) {
        case 'R': return "Running";
        case 'S': return "Sleeping (interruptible)";
        case 'D': return "Sleeping (uninterruptible)";
        case 'T': return "Stopped";
        case 't': return "Tracing stop";
        case 'Z': return "Zombie";
        case 'X': return "Dead";
        case 'x': return "Dead";
        case 'K': return "Wakekill";
        case 'W': return "Waking";
        case 'P': return "Parked";
        case '?': return "Unknown";
        default: return "Undefined";
    }
}


// Function to get process state from /proc/[pid]/stat
char get_process_state(pid_t pid) {
    char filename[64];
    char line[MAX_LINE];
    char state;
    FILE *file;
    
    snprintf(filename, sizeof(filename), "/proc/%d/stat", pid);
    file = fopen(filename, "r");
    
    if (file == NULL) {
        return '?'; // Process not found or no permission
    }
    
    if (fgets(line, sizeof(line), file) != NULL) {
        // The state is the third field in /proc/[pid]/stat
        char *token = strtok(line, " ");
        token = strtok(NULL, " "); // Skip PID
        token = strtok(NULL, " "); // Skip command
        //token = strtok(NULL, " "); // This is the state
        puts(line);
        
        if (token != NULL) {
            state = token[0];
        } else {
            state = '?';
        }
    } else {
        state = '?';
    }
    
    fclose(file);
    return state;
}

void print_scheduling_info() {
    pid_t pid = getpid();
    int policy;
    struct sched_param param;
    int nice_val;
    struct rusage usage;
    
    // Get scheduling policy and parameters
    policy = sched_getscheduler(pid);
    if (policy == -1) {
        perror("sched_getscheduler");
        return;
    }
    
    if (sched_getparam(pid, &param) == -1) {
        perror("sched_getparam");
        return;
    }
    
    // Get nice value
    errno = 0;
    nice_val = getpriority(PRIO_PROCESS, pid);
    if (errno != 0) {
        perror("getpriority");
    }
    
    // Get resource usage
    if (getrusage(RUSAGE_SELF, &usage) == -1) {
        perror("getrusage");
        return;
    }
    
    // Get current time
    time_t now = time(NULL);
    struct tm *local_time = localtime(&now);
    char time_str[64];
    strftime(time_str, sizeof(time_str), "%Y-%m-%d %H:%M:%S", local_time);
    
    printf("\n================== SCHEDULING INFO (CPU-BOUND) ==================\n");
    printf("Timestamp: %s\n", time_str);
    printf("PID: %d\n", pid);
    printf("State: %s\n", state_to_string(get_process_state(pid)));
    printf("Policy: %s (%d)\n", get_policy_name(policy), policy);
    printf("RT Priority: %d\n", param.sched_priority);
    printf("Nice Value: %d\n", nice_val);
    
    // Print priority ranges for current policy
    int min_prio = sched_get_priority_min(policy);
    int max_prio = sched_get_priority_max(policy);
    if (min_prio != -1 && max_prio != -1) {
        printf("Priority Range: %d - %d\n", min_prio, max_prio);
    }
    
    // Print CPU usage statistics
    double user_time = usage.ru_utime.tv_sec + usage.ru_utime.tv_usec / 1000000.0;
    double sys_time = usage.ru_stime.tv_sec + usage.ru_stime.tv_usec / 1000000.0;
    
    printf("CPU Time (User): %.2f seconds\n", user_time);
    printf("CPU Time (System): %.2f seconds\n", sys_time);
    printf("CPU Time (Total): %.2f seconds\n", user_time + sys_time);
    
    printf("Context Switches (Voluntary): %ld\n", usage.ru_nvcsw);
    printf("Context Switches (Involuntary): %ld\n", usage.ru_nivcsw);
    printf("Memory Usage (Max RSS): %ld KB\n", usage.ru_maxrss);
    
    // Print additional scheduling information from /proc/self/sched
    FILE *sched_file = fopen("/proc/self/sched", "r");
    if (sched_file) {
        printf("\n--- Additional Scheduler Information ---\n");
        char line[256];
        int line_count = 0;
        while (fgets(line, sizeof(line), sched_file) && line_count < 15) {
            // Print interesting scheduler statistics
            if (strstr(line, "policy") || 
                strstr(line, "prio") || 
                strstr(line, "nice") ||
                strstr(line, "exec_runtime") ||
                strstr(line, "vruntime") ||
                strstr(line, "sum_exec_runtime") ||
                strstr(line, "switches") ||
                strstr(line, "wait_count")) {
                printf("%s", line);
                line_count++;
            }
        }
        fclose(sched_file);
    }
    
    printf("=====================================================\n");
}

void print_system_scheduler_info() {
    printf("\n================ SYSTEM SCHEDULER INFO ==============\n");
    
    // Print RT throttling information
    FILE *rt_runtime = fopen("/proc/sys/kernel/sched_rt_runtime_us", "r");
    FILE *rt_period = fopen("/proc/sys/kernel/sched_rt_period_us", "r");
    
    if (rt_runtime && rt_period) {
        int runtime, period;
        fscanf(rt_runtime, "%d", &runtime);
        fscanf(rt_period, "%d", &period);
        
        printf("RT Throttling:\n");
        printf("  RT Runtime: %d μs\n", runtime);
        printf("  RT Period: %d μs\n", period);
        printf("  RT Bandwidth: %.1f%%\n", (double)runtime / period * 100);
        
        fclose(rt_runtime);
        fclose(rt_period);
    }
    
    // Print scheduler features (if available)
    FILE *sched_features = fopen("/sys/kernel/debug/sched_features", "r");
    if (sched_features) {
        printf("Scheduler Features: ");
        char features[512];
        if (fgets(features, sizeof(features), sched_features)) {
            printf("%s", features);
        }
        fclose(sched_features);
    }
    
    // Print load average
    double loadavg[3];
    if (getloadavg(loadavg, 3) != -1) {
        printf("Load Average: %.2f %.2f %.2f\n", loadavg[0], loadavg[1], loadavg[2]);
    }
    
    printf("====================================================\n");
}


// Function to initialize matrix with random values
void randomize_matrix(double matrix[SIZE][SIZE]) {
    int i, j;
    for (i = 0; i < SIZE; i++) {
        for (j = 0; j < SIZE; j++) {
            // Generate random values between 0 and 10
            matrix[i][j] = ((double)rand() / RAND_MAX) * 10.0;
        }
    }
}

void do_some_work() {
   double a[SIZE][SIZE], b[SIZE][SIZE], c[SIZE][SIZE];
   int i, j, k;

   randomize_matrix(a);
   randomize_matrix(b);
   // Initialize result matrix to zero
   for (i = 0; i < SIZE; i++) {
       for (j = 0; j < SIZE; j++) {
           c[i][j] = 0.0;
       }
   }
    
   // Perform matrix multiplication: C = A * B
   for (i = 0; i < SIZE; i++) {
       for (j = 0; j < SIZE; j++) {
           for (k = 0; k < SIZE; k++) {
               c[i][j] += a[i][k] * b[k][j];
           }
       }
   }


/*
    // Simulate some CPU work

    volatile long counter = 0;
    for (int i = 0; i < 1000000; i++) {
        counter += i;
    }
    
    // Simulate some I/O work
    if (rand() % 10 == 0) {
        usleep(10000); // 10ms sleep occasionally
    }
*/
}

void print_usage(const char *prog_name) {
    printf("Usage: %s [options]\n", prog_name);
    printf("Options:\n");
    printf("  -p policy   Set scheduling policy (other|fifo|rr|batch|idle)\n");
    printf("  -r priority Set RT priority (1-99, for FIFO/RR policies)\n");
    printf("  -n nice     Set nice value (-20 to 19, for OTHER/BATCH policies)\n");
    printf("  -i interval Update interval in seconds (default: 5)\n");
    printf("  -s          Show system scheduler info on startup\n");
    printf("  -h          Show this help message\n");
    printf("\nExamples:\n");
    printf("  %s                    # Run with default SCHED_OTHER\n", prog_name);
    printf("  %s -p fifo -r 50      # Run with SCHED_FIFO, priority 50\n", prog_name);
    printf("  %s -p other -n 10     # Run with SCHED_OTHER, nice 10\n", prog_name);
    printf("  %s -i 2 -s            # Update every 2 seconds, show system info\n", prog_name);
}

int main(int argc, char *argv[]) {
    int interval = 5;
    int show_system_info = 0;
    int policy = SCHED_OTHER;
    int rt_priority = 1;
    int nice_value = 0;
    int set_policy = 0;
    int set_nice = 0;
    
    // Parse command line arguments
    int opt;
    while ((opt = getopt(argc, argv, "p:r:n:i:sh")) != -1) {
        switch (opt) {
            case 'p':
                set_policy = 1;
                if (strcmp(optarg, "other") == 0) {
                    policy = SCHED_OTHER;
                } else if (strcmp(optarg, "fifo") == 0) {
                    policy = SCHED_FIFO;
                } else if (strcmp(optarg, "rr") == 0) {
                    policy = SCHED_RR;
                } else if (strcmp(optarg, "batch") == 0) {
                    policy = SCHED_BATCH;
                } else if (strcmp(optarg, "idle") == 0) {
                    policy = SCHED_IDLE;
                } else {
                    fprintf(stderr, "Invalid policy: %s\n", optarg);
                    print_usage(argv[0]);
                    exit(1);
                }
                break;
            case 'r':
                rt_priority = atoi(optarg);
                if (rt_priority < 1 || rt_priority > 99) {
                    fprintf(stderr, "RT priority must be between 1 and 99\n");
                    exit(1);
                }
                break;
            case 'n':
                set_nice = 1;
                nice_value = atoi(optarg);
                if (nice_value < -20 || nice_value > 19) {
                    fprintf(stderr, "Nice value must be between -20 and 19\n");
                    exit(1);
                }
                break;
            case 'i':
                interval = atoi(optarg);
                if (interval < 1) {
                    fprintf(stderr, "Interval must be at least 1 second\n");
                    exit(1);
                }
                break;
            case 's':
                show_system_info = 1;
                break;
            case 'h':
                print_usage(argv[0]);
                exit(0);
            default:
                print_usage(argv[0]);
                exit(1);
        }
    }
    
    // Set up signal handlers
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);
    
    printf("Scheduling Monitor - PID: %d\n", getpid());
    printf("Press Ctrl+C to stop gracefully\n");
    printf("Update interval: %d seconds\n", interval);
    
    // Set scheduling policy if requested
    if (set_policy) {
        struct sched_param param;
        
        if (policy == SCHED_FIFO || policy == SCHED_RR) {
            param.sched_priority = rt_priority;
        } else {
            param.sched_priority = 0;
        }
        
        if (sched_setscheduler(0, policy, &param) == -1) {
            perror("Failed to set scheduling policy");
            printf("Note: RT policies require root privileges\n");
        } else {
            printf("Successfully set scheduling policy to %s\n", get_policy_name(policy));
        }
    }
    
    // Set nice value if requested
    if (set_nice) {
        if (setpriority(PRIO_PROCESS, 0, nice_value) == -1) {
            perror("Failed to set nice value");
        } else {
            printf("Successfully set nice value to %d\n", nice_value);
        }
    }
    
    // Show system scheduler info if requested
    if (show_system_info) {
        print_system_scheduler_info();
    }
    
    // Main monitoring loop
    int iteration = 0;
    while (keep_running) {
        iteration++;
        printf("\n[Iteration %d]\n", iteration);
        
        print_scheduling_info();
        
        // Do some work between updates
        for (int i = 0; i < interval && keep_running; i++) {
            do_some_work();
            sleep(1);
        }
    }
    
    printf("\nScheduling monitor stopped after %d iterations.\n", iteration);
    printf("Final scheduling information:\n");
    print_scheduling_info();
    
    return 0;
}
