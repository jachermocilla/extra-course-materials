#include <stdio.h>
#include <stdlib.h>

typedef unsigned char *pointer;

void show_bytes(pointer start, int len){
   int i, c, k;

   for (i=0;i<len;i++){
      printf("0x%.2X ",start[i]);
   }
   printf(", ");
   
   //binary
   for (i=0;i<len;i++){
      for (c = 7; c >= 0; c--){
         k = start[i] >> c;
         if (k & 1)
            printf("1");
         else
            printf("0");
      }
      printf(" ");
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
   show_bytes((pointer)&lp,sizeof(long *));

   printf("int *ip = &b;\n");
   show_bytes((pointer)&ip,sizeof(int *));
}

