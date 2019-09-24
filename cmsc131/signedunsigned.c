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
   int tx=100, ty;
   unsigned ux, uy=-100;
   int x=-1;
   int u=2147483648;

   ux=tx;  
 
   show_bytes((pointer)&tx,sizeof(int));
   show_bytes((pointer)&uy,sizeof(unsigned));

   printf("x = %x = %d\n", x, x);
   printf("u = %x = %d\n", u, u);
   
}

