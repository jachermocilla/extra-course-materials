//https://www.usna.edu/Users/cs/roche/courses/s19ic221/lab05.html
/* memleak_example.c */
//valgrind --leak-check=full ./memleak.elf
#include <stdio.h>
#include <stdlib.h>

int main(int argc, char *argv[]) {

   int *a = malloc(sizeof(int));

   *a = 10;

   printf("%d\n", *a);

   a = calloc(3, sizeof(int));
   a[0] = 10;
   a[1] = 20;
   a[2] = 30;

   printf("%d %d %d\n", a[0], a[1], a[2]);

   return 0;
}
