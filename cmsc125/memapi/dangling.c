#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char *argv[]) {
   int *p = (int *) malloc(sizeof(int));
   free(p);
   *p = 4;
   printf("%d\n",*p);
}
