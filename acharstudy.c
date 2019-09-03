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

   //the case for zero
   sca=0;
   scb=~0;
   sctmp=scb+1;
   show_bytes((pointer)&sca,sizeof(signed char ));
   show_bytes((pointer)&scb,sizeof(signed char ));
   show_bytes((pointer)&sctmp,sizeof(signed char ));

   printf("\n");

   //addition of unsigned
   uca=0xFE;
   ucb=0x3;
   uctmp = uca + ucb;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

   //equivalent to getting the (x+y) mod 2^w 
   uctmp = (uca + ucb) % (1 << 8);
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

   printf("\n");

   //addition of signed
   sca=60;
   scb=-80;
   sctmp=sca+scb;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));
   show_bytes((pointer)&sctmp,sizeof(char));
  
   //signed addition using properties in relation to unsigned 
   uca = sca;
   ucb = scb;
   uctmp = uca+ucb;
   sctmp = uctmp;
   show_bytes((pointer)&sctmp,sizeof(char));
  
   printf("\n");
   //unsigned multiplication 
   uca=0x9B;
   ucb=0x5;
   uctmp = uca * ucb; 
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));
   
   //equivalent to getting the (x*y) mod 2^w 
   uctmp = (uca * ucb) % (1 << 8);
   show_bytes((pointer)&uctmp,sizeof(unsigned char));
   
   printf("\n");
   
   //signed multiplication 
   sca=-50;
   scb=5;
   sctmp = sca * scb; 
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));
   show_bytes((pointer)&sctmp,sizeof(char));
   
   //equivalent to getting the (x*y) mod 2^w 
   sctmp = (sca * scb) % (1 << 8);
   show_bytes((pointer)&sctmp,sizeof(char));
   
   printf("\n");

   //power of two multiply with shift 3 * 2^4
   uca= 3 << 4;
   ucb = 3 * (2  * 2 * 2 * 2 );
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
  
   //signed 
   sca= -3 << 4;
   scb = -3 * (2  * 2 * 2 * 2 );
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));

   


}

