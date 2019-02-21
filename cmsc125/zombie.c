#include <stdlib.h> 
#include <sys/types.h> 
#include <unistd.h> 
int main() 
{ 
   pid_t pid = fork(); 
  
   // Parent process  
   if (pid > 0){ 
        sleep(60);
   }
   
   // Child process 
   else{
      execlp("./child.exe","child.exe",NULL);
   } 
   return 0; 
} 

