//Build the executable for this first  
//because it will be loaded by the parent 
//process
//
// $gcc -o child.elf child.c

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <unistd.h>
#include "common.h"


int main(int argc, char *argv[]){
   char name[64];
   int lifetime;

   if (argc < 3){
      printf("./child <name> <lifetime>.\n");
      exit(1);
   }

   strcpy(name,argv[1]);
   lifetime = atoi(argv[2]);
   printf("Child: I am %s. My Lifetime=%d seconds. pid: %d \n",name,lifetime,getpid());
   Spin(lifetime);
   printf("Child: %s died.\n",name);
   return 0;
}
