/*
 * Code from textbook
 *
 */

#include <sys/types.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#define BUFFER_SIZE 25
#define READ_END 0
#define WRITE_END 1

int main(void){
   /*Message to send*/
   char write_msg[BUFFER_SIZE] = "Greetings";

   /*Placeholder for the message to receive*/
   char read_msg[BUFFER_SIZE];

   /*File desciptors for the pipe*/
   int fd[2];

   /*Process ID */
   pid_t pid;

   /* create the pipe, note that this pipe was created in the parent process */
   if (pipe(fd) == -1) {
      fprintf(stderr,"Pipe failed");
      return 1;
   }

   /* fork a child process */
   pid = fork();
   if (pid < 0) { /* error occurred */
      fprintf(stderr, "Fork Failed");
      return 1;
   }

   if (pid == 0) { /* child process */

      /* close the unused end of the pipe */
      close(fd[READ_END]);

      printf("Child: wrote '%s'\n",write_msg);

      /* write to the pipe */
      write(fd[WRITE_END], write_msg, strlen(write_msg)+1);

      /* close the write end of the pipe */
      close(fd[WRITE_END]);
   
   }
   else { /* parent process */

      /* close the unused end of the pipe */
      close(fd[WRITE_END]);

      /* read from the pipe */
      read(fd[READ_END], read_msg, BUFFER_SIZE);

      /* close the write end of the pipe */
      close(fd[READ_END]);

      printf("Parent: read '%s'\n",read_msg);
   }

   return 0;
}
