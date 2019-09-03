/** 
 * This code demonstrates the intricacies of intergral types 
 * using char (8 bits) and short (16 bits) both signed and 
 * unsigned. 
 * Based on the slides for Chapter 2 of CS:APP2e
 *
 * Enjoy! :)
 * 
 */

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


   printf("//shows the correspondence between signed and unsigned\n"); 
   for (uc=0; uc<UCHAR_MAX; uc++){
      sc = uc;
   
      //print the actual values
      printf("%d %d\n",uc,sc);

      //print the internal representation
      show_bytes((pointer)&uc,sizeof(unsigned char));
      show_bytes((pointer)&sc,sizeof(char));
      printf("\n\n");
   }

   printf("//expansion unsigned\n");
   usa = uca;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&usa,sizeof(unsigned short));
   
   printf("\n");
   
   printf("//expansion signed\n");
   ssa = sca;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&ssa,sizeof(short));

   printf("\n");


   printf("//truncate unsigned\n");
   usa = 260;
   uca = usa; 
   show_bytes((pointer)&usa,sizeof(unsigned short));
   show_bytes((pointer)&uca,sizeof(unsigned char ));

   printf("//truncation is equivalent to getting the mod 2^w\n");
   unsigned short ustmp = usa % (1 << 8);
   show_bytes((pointer)&ustmp,sizeof(unsigned short));
   
   printf("\n");

   
   printf("//truncate signed\n");
   ssa = -260;
   sca = ssa; 
   show_bytes((pointer)&ssa,sizeof(signed short));
   show_bytes((pointer)&sca,sizeof(signed char ));
   
   printf("//truncation is equivalent to getting the mod 2^w but treated as signed\n");
   short sstmp = ssa % (1 << 8);
   show_bytes((pointer)&sstmp,sizeof(short));
   
   printf("\n");

   printf("//complement and increment\n");

   sca=0x6D;
   scb=~sca;
   sctmp=scb+1;
   show_bytes((pointer)&sca,sizeof(signed char ));
   show_bytes((pointer)&scb,sizeof(signed char ));
   show_bytes((pointer)&sctmp,sizeof(signed char ));

   printf("//the case for zero\n");
   sca=0;
   scb=~0;
   sctmp=scb+1;
   show_bytes((pointer)&sca,sizeof(signed char ));
   show_bytes((pointer)&scb,sizeof(signed char ));
   show_bytes((pointer)&sctmp,sizeof(signed char ));

   printf("\n");

   printf("//addition of unsigned\n");
   uca=0xFE;
   ucb=0x3;
   uctmp = uca + ucb;
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

   printf("//equivalent to getting the (x+y) mod 2^w \n");
   uctmp = (uca + ucb) % (1 << 8);
   show_bytes((pointer)&uctmp,sizeof(unsigned char));

   printf("\n");

   printf("//addition of signed\n");
   sca=60;
   scb=-80;
   sctmp=sca+scb;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));
   show_bytes((pointer)&sctmp,sizeof(char));
  
   printf("//signed addition using properties in relation to unsigned\n");
   uca = sca;
   ucb = scb;
   uctmp = uca+ucb;
   sctmp = uctmp;
   show_bytes((pointer)&sctmp,sizeof(char));
  
   printf("\n");
   
   printf("//unsigned multiplication\n");
   uca=0x9B;
   ucb=0x5;
   uctmp = uca * ucb; 
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   show_bytes((pointer)&uctmp,sizeof(unsigned char));
   
   printf("//equivalent to getting the (x*y) mod 2^w \n");
   uctmp = (uca * ucb) % (1 << 8);
   show_bytes((pointer)&uctmp,sizeof(unsigned char));
   
   printf("\n");
   
   printf("//signed multiplication\n"); 
   sca=-50;
   scb=5;
   sctmp = sca * scb; 
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));
   show_bytes((pointer)&sctmp,sizeof(char));
   
   printf("//equivalent to getting the (x*y) mod 2^w \n");
   sctmp = (sca * scb) % (1 << 8);
   show_bytes((pointer)&sctmp,sizeof(char));
   
   printf("\n");

   printf("//power of two multiply with shift 3 * 2^4\n");
   uca= 3 << 4;
   ucb = 3 * (2  * 2 * 2 * 2 );
   show_bytes((pointer)&uca,sizeof(unsigned char));
   show_bytes((pointer)&ucb,sizeof(unsigned char));
   
   printf("\n");
  
   printf("//power of two multiply with shift -3 * 2^4\n");
   sca= -3 << 4;
   scb = -3 * (2  * 2 * 2 * 2 );
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));
   
   printf("\n");

   printf("//signed power of two divide with shift\n");
   //-66 / (2^4) = 66 >> 4

   sca = -66 / (2*2*2*2);
   scb = -66 >> 4;
   sctmp = (-66 + (1 << 4) - 1) >> 4;
   printf("%d\n",sca);;
   show_bytes((pointer)&sca,sizeof(char));
   show_bytes((pointer)&scb,sizeof(char));  //wrong since -66 < 9
   show_bytes((pointer)&sctmp,sizeof(char));  //correct
      


}

