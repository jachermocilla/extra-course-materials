#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <unistd.h>
#include "common.h"

int main(){
   pid_t pid0,pid1;
   int status;
   

   printf("Parent: pid: %d\n",getpid());
/* fork a child process */
   pid0 = fork();
   if (pid0 < 0) { /* error occurred */
      fprintf(stderr, "Fork Failed");
      return 1;
   }

   if (pid0 == 0) { /* child process, child0 */
      //this child process is just the same as the parent
      printf("Child: child0 pid: %d\n",getpid());
      Spin(50);
      printf("Child: child0 died \n");
   } else {
      pid1 = fork();
      if (pid1 < 0) { /* error occurred */
         fprintf(stderr, "Fork Failed");
         return 1;
      }
      if (pid1 == 0) { /* child process, child1 */
         execlp("./child.elf","child.elf","child1","40",NULL);
      }
      //child1 will never reach this
      /* parent will wait for the child to complete */
      printf("Parent: Waiting for child1 to die.\n");
      waitpid(pid1,&status,0);
   }
   return 0;
}
