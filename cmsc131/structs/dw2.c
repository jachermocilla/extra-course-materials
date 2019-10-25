#include <stdio.h>
#include <stdlib.h>


union {
   unsigned char c[8];
   unsigned short s[4];
   unsigned int i[2];
   unsigned long l[1];
}dw;


int main(){
   int j;
   for (j = 0; j < 8; j++)
      dw.c[j] = 0xf0 + j;

   printf("Characters 0-7 == [0x%x,0x%x,0x%x,0x%x,0x%x,0x%x,0x%x,0x%x]\n",
            dw.c[0], dw.c[1], dw.c[2], dw.c[3],
            dw.c[4], dw.c[5], dw.c[6], dw.c[7]);
   printf("Shorts 0-3 == [0x%x,0x%x,0x%x,0x%x]\n", dw.s[0], dw.s[1], dw.s[2], dw.s[3]);
   printf("Ints 0-1 == [0x%x,0x%x]\n", dw.i[0], dw.i[1]);
   printf("Long 0 == [0x%lx]\n",dw.l[0]);
   return 0;
}
