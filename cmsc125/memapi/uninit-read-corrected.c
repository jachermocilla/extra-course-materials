#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(){
   int *x = (int *) malloc(sizeof(int)); // allocated
   *x = 4; 
   printf("*x = %d n", *x); // uninitialized memory access
   free(x);
}