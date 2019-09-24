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
   unsigned short usx = sx;
   int x = sx;
   unsigned ux = usx;

   printf("sx= %d\n",sx);
   show_bytes((pointer)&sx,sizeof(short));
   printf("sx= %u\n",usx);
   show_bytes((pointer)&usx,sizeof(unsigned short));
   printf("x= %d\n",x);
   show_bytes((pointer)&x,sizeof(int));
   printf("ux= %u\n",ux);
   show_bytes((pointer)&ux,sizeof(unsigned));
}

