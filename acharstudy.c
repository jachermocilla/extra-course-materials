#include <stdio.h>
#include <stdlib.h>
#include <limits.h>

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
   unsigned char uc;
   char sc;

   for (uc=0; uc<UCHAR_MAX; uc++){
      sc = uc;
   
      //print the actual values
      printf("%d %d\n",uc,sc);

      //print the internal representation
      show_bytes((pointer)&uc,sizeof(unsigned char));
      show_bytes((pointer)&sc,sizeof(char));

      printf("\n\n");
   }

}

//It can be observed that the internal representations are the same
//but the interpretations are different for signed and unsigned
