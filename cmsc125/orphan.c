#include<stdio.h> 
#include <sys/types.h> 
#include <unistd.h> 

int main() 
{ 
   // Create a child process   
   pid_t pid = fork(); 

   if (pid > 0){ 
      printf("Parent: I am alive but I'll die before my child.\n"); 
      sleep(20);
      printf("Parent: I died. Child becomes orphan.\n"); 
   }

   // Note that pid is 0 in child process 
   // and negative if fork() fails 
   else if (pid == 0) 
   { 
      execlp("./child.exe","child.exe",NULL);
   } 

   return 0; 
} 

