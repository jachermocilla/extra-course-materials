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
   int a=15213;
   int b=-15213;
   long c=15213;
   long *lp=&c;
   int *ip=&b;


   printf("int a = 15213;\n");
   show_bytes((pointer)&a,sizeof(int));

   printf("int b = -15213;\n");
   show_bytes((pointer)&b,sizeof(int));

   printf("long c = 15213;\n");
   show_bytes((pointer)&c,sizeof(long));
   
   printf("long *lp = &c;\n");
   show_bytes((pointer)&c,sizeof(long *));

   printf("int *ip = &b;\n");
   show_bytes((pointer)&c,sizeof(int *));
}

