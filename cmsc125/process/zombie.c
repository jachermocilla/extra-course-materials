//Zombie processes are those that have died but 
//the parent process has not yet harvested the 
//status code of the child process. 

#include <stdlib.h> 
#include <sys/types.h> 
#include <unistd.h> 
#include <stdio.h>
#include "common.h"

int main() 
{ 
   pid_t pid = fork(); 
  
   // Parent process  
   if (pid > 0){ 
      printf("Parent: pid: %d\n",getpid());  
      Spin(60); //sleep longer than child so that the child dies first
   }
   
   // Child process 
   else{
      execlp("./child.elf","child.elf","child","40",NULL);
   } 
   return 0; 
} 

