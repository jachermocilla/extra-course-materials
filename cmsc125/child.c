//Build the executable for this first  
//because it will be loaded by the parent 
//process
//
// $gcc -o child.exe child.c

#include <stdio.h>
#include <unistd.h>

int main(){
   printf("Child: I am alive. My Lifetime=40 seconds.\n");
   sleep(40);
   printf("Child: I died.\n");
   return 0;
}
