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
   int x = 53191;
   short sx = (short) x;
   int y = sx;


   printf("x = 53191;\n");
   show_bytes((pointer)&x,sizeof(int));

   printf("sx = %d;\n",sx);
   show_bytes((pointer)&sx,sizeof(short));

   printf("y = %d;\n",y);
   show_bytes((pointer)&y,sizeof(int));
   
}

