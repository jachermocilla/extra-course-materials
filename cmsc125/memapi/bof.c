#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(){
   char *src = "Hello";
   char *dst;
   dst = (char *)malloc(strlen(src));
   strcpy(dst,src);
   printf("%s\n",dst);
   free(dst);
   return 0;
}
