//Zombie processes are those that have died but 
//the parent process has not yet harvested the 
//status code of the child process. 

#include <stdlib.h> 
#include <sys/types.h> 
#include <unistd.h> 
int main() 
{ 
   pid_t pid = fork(); 
  
   // Parent process  
   if (pid > 0){ 
        sleep(60); //sleep longer that child so that the child dies first
   }
   
   // Child process 
   else{
      execlp("./child.exe","child.exe",NULL);
   } 
   return 0; 
} 

