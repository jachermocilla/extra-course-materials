#include <stdlib.h> 
#include <sys/types.h> 
#include <unistd.h> 
#include <stdio.h>
#include "common.h"

int main() 
{ 
   pid_t pid = fork(); 
   return pid;
}
