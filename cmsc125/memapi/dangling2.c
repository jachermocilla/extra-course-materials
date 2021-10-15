#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char *argv[]) {
   int *p;
   {
      int a=4;
      p=&a;
   }
   *p=5;
}
