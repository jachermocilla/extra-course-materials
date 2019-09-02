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
   short ssa;


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
   ssa = sca;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&ssa,sizeof(short));

   printf("\n");


   //truncate unsigned
   usa = 260;
   uca = usa; 
   show_bytes((pointer)&usa,sizeof(unsigned short));
   show_bytes((pointer)&uca,sizeof(unsigned char ));

   //truncation is equivalent to getting the mod 2^w
   unsigned short ustmp = usa % (1 << 8);
   show_bytes((pointer)&ustmp,sizeof(unsigned short));
   
   printf("\n");

   
   //truncate signed
   ssa = -260;
   sca = ssa; 
   show_bytes((pointer)&ssa,sizeof(signed short));
   show_bytes((pointer)&sca,sizeof(signed char ));
   
   //truncation is equivalent to getting the mod 2^w but treated as signed
   short sstmp = ssa % (1 << 8);
   show_bytes((pointer)&sstmp,sizeof(short));
   
   printf("\n");

   //complement and increment

   sca=0x6D;
   scb=~sca;
   sctmp=scb+1;
   show_bytes((pointer)&sca,sizeof(signed char ));
   show_bytes((pointer)&scb,sizeof(signed char ));
   show_bytes((pointer)&sctmp,sizeof(signed char ));

   sca=0;
   scb=~0;
   sctmp=scb+1;
   show_bytes((pointer)&sca,sizeof(signed char ));
   show_bytes((pointer)&scb,sizeof(signed char ));
   show_bytes((pointer)&sctmp,sizeof(signed char ));

   printf("\n");

   //addition of signed       
   uctmp = uca + ucb;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

}

