#include <stdio.h>
#include <stdlib.h>

typedef unsigned char *pointer;

void show_bytes(pointer start, int len){
   int i;
   for (i=0;i<len;i++){
      printf("%p\t0x%.2x\n",start+i,start[i]);
   }
   printf("\n");
}

int main(){
   short sx = -12345;
   unsigned uy = sx;

   printf("uy = %u\n", uy);
   show_bytes((pointer)&uy,sizeof(unsigned));
}

