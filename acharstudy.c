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
   unsigned char uca=254,ucb=15;
   unsigned char uctmp;
   char sca=-100,scb=-15;
   char sctmp;
   unsigned short usa;
   short sa;


   //shows the correspondence between signed and unsigned 
   for (uc=0; uc<UCHAR_MAX; uc++){
      sc = uc;
   
      //print the actual values
      printf("%d %d\n",uc,sc);

      //print the internal representation
      show_bytes((pointer)&uc,sizeof(unsigned char));
      show_bytes((pointer)&sc,sizeof(char));
      printf("\n\n");
   }

   //expansion unsigned
   usa = uca;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&usa,sizeof(unsigned short));
   
   printf("\n");
   
   //expansion signed
   sa = sca;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&sa,sizeof(short));

   printf("\n");

   //addition of signed       
   uctmp = uca + ucb;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

}

//It can be observed that the internal representations are the same
//but the interpretations are different for signed and unsigned
