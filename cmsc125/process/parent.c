#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <unistd.h>

int main(){
   pid_t pid;
   /* fork a child process */
   pid = fork();
   if (pid < 0) { /* error occurred */
      fprintf(stderr, "Fork Failed");
      return 1;
   }
   else if (pid == 0) { /* child process */
      execlp("./child.exe","child.exe",NULL);
   }
   else { /* parent process */
      /* parent will wait for the child to complete */
      printf("Parent: Waiting for my child to die.\n");
      wait(NULL);
      printf("Parent: Child died.\n");
   }
   return 0;
}
