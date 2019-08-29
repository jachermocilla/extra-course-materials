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
   short a=12345;
   short ma= -a;

   show_bytes((pointer)&a,sizeof(short));

   show_bytes((pointer)&ma,sizeof(short));
}

