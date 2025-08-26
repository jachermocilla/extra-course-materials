/**

 $gcc -g -o psgrep.elf psgrep.c
 $gdb -tui ./psgrep.elf
 gdb>set follow-fork-mode child
 gdb>set detach-on-fork off
 gdb>info inferiors #check gdb controlled processes
 gdb>inferior 1 #go to a different process





*/
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>
#include <string.h>

int main() {
    int pipefd[2];  // File descriptors for the pipe
    pid_t pid1, pid2;
    int status;
    
    // Create the pipe
    if (pipe(pipefd) == -1) {
        perror("pipe");
        exit(EXIT_FAILURE);
    }
    
    // Fork the first child process for "ps -A"
    pid1 = fork();
    if (pid1 == -1) {
        perror("fork");
        exit(EXIT_FAILURE);
    }
    
    if (pid1 == 0) {
        // First child process - execute "ps -A"
        
        // Close the read end of the pipe (we only write)
        close(pipefd[0]);
        
        // Redirect stdout to the write end of the pipe
        if (dup2(pipefd[1], STDOUT_FILENO) == -1) {
            perror("dup2");
            exit(EXIT_FAILURE);
        }
        
        // Close the write end after duplication
        close(pipefd[1]);
        
        // Execute ps -A
        execl("/bin/ps", "ps", "-A", NULL);
        
        // If execl fails
        perror("execl ps");
        exit(EXIT_FAILURE);
    }
    
    // Fork the second child process for "grep bash"
    pid2 = fork();
    if (pid2 == -1) {
        perror("fork");
        exit(EXIT_FAILURE);
    }
    
    if (pid2 == 0) {
        // Second child process - execute "grep bash"
        
        // Close the write end of the pipe (we only read)
        close(pipefd[1]);
        
        // Redirect stdin to the read end of the pipe
        if (dup2(pipefd[0], STDIN_FILENO) == -1) {
            perror("dup2");
            exit(EXIT_FAILURE);
        }
        
        // Close the read end after duplication
        close(pipefd[0]);
        
        // Execute grep bash
        execl("/bin/grep", "grep", "bash", NULL);
        
        // If execl fails
        perror("execl grep");
        exit(EXIT_FAILURE);
    }
    
    // Parent process
    
    // Close both ends of the pipe in the parent
    // (the children have their own copies)
    close(pipefd[0]);
    close(pipefd[1]);
    
    // Wait for both child processes to complete
    waitpid(pid1, &status, 0);
    if (WIFEXITED(status)) {
        printf("ps process exited with status: %d\n", WEXITSTATUS(status));
    }
    
    waitpid(pid2, &status, 0);
    if (WIFEXITED(status)) {
        printf("grep process exited with status: %d\n", WEXITSTATUS(status));
    }
    
    return 0;
}
