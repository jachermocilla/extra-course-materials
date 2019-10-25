#include <stdio.h>
#include <stdlib.h>


union {
   unsigned char c[8];
   unsigned short s[4];
   unsigned int i[2];
   unsigned long l[1];
}dw;

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
   dw.c[0]=0x8;
   dw.c[1]=0x9;
   dw.c[2]=0xA;
   dw.c[3]=0xB;
   dw.c[4]=0xC;
   dw.c[5]=0xD;
   dw.c[6]=0xE;
   dw.c[7]=0xF;

   show_bytes(dw.c,sizeof(char)*8);
   show_bytes((pointer)dw.s,sizeof(short)*4);
   show_bytes((pointer)dw.i,sizeof(int)*2);
   show_bytes((pointer)dw.l,sizeof(long)*1);


}
