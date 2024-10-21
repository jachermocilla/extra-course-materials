#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(){
   int *x = (int *) malloc(sizeof(int)); 
   printf("*x = %d \n", *x); 
   free(x);
}
