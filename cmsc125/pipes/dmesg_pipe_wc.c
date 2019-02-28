/*
 * This example code illustrates how the command 'dmesg | wc' 
 * can be implemented using pipes.
 * 
 * All processes in linux has the following initial file descriptors:
 * STDIN_FILENO = 0 which is the keyboard
 * STDOUT_FILENO = 1 which is the screen
 * STDERR_FILENO = 2 which is the screen
 * 
 * $gcc -o dmesg_pipe_wc.exe dmesg_pipe_wc.c
 * $./dmesg_pipe_wc.exe
 *
 * by jchermocilla@up.edu.ph
 */

#include <sys/types.h>
#include <sys/wait.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>

#define READ_END 0      
#define WRITE_END 1

int main(void){

   /*File desciptors for the pipe*/
   int fd[2];

   /*Process ID */
   pid_t pid_dmesg;
   pid_t pid_wc;
   
   int status;


   /* create the pipe, note that this pipe was created in the parent process */
   if (pipe(fd) == -1) {
      fprintf(stderr,"Pipe failed");
      return 1;
   }

   /* fork a child process for 'dmesg' */
   pid_dmesg = fork();


   if (pid_dmesg > 0) { /* parent process */


      //fork a child process for 'wc'
      pid_wc = fork();
      
      if (pid_wc > 0){
         //Parent waits for the children
         wait(&status);
      }

      else{ /*Child process for 'wc' */
             
         /* close the unused end of the pipe */
         close(fd[WRITE_END]);

         /* Set the STDIN of the 'wc' process to the read end of the pipe */
         dup2(fd[READ_END],STDIN_FILENO);

         /*Load the image for 'wc'*/
         execlp("/usr/bin/wc","wc",NULL);
         
         /* close the read end of the pipe */
         close(fd[READ_END]);

      }
   }
   else { /* child process for dmesg */
         
         /* close the unused end of the pipe */
         close(fd[READ_END]);

         /* Set the STDOUT of the 'wc' process to the write end of the pipe */
         dup2(fd[WRITE_END],STDOUT_FILENO); 
         
         /* Load the image for 'dmesg; */
         execlp("/bin/dmesg","dmesg",NULL);

         /* close the write end of the pipe */
         close(fd[WRITE_END]);

      }
   return 0;
}
