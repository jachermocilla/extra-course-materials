#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(){
   char *src = "Hello";
   char *dst;
   dst = (char *)malloc(strlen(src)+1);
   strcpy(dst,src);
   free(dst);
   return 0;
}