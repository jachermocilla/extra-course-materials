#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

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

void show_bin(pointer start, int len){
   int i, c, k;
   unsigned char byte;

   //binary
   for (i=0;i<len;i++){
      byte = start[i];
      for (c = 7; c >= 0; c--){
         k = byte >> c;
         if (k & 1)
            printf("1");
         else
            printf("0");
      }
      printf(" ");
   }
   printf(" ");
}

void show_float(pointer start){
   int i, c, k;

   show_bin((pointer)&start[i+3],sizeof(unsigned char));
   show_bin((pointer)&start[i+2],sizeof(unsigned char));
   show_bin((pointer)&start[i+1],sizeof(unsigned char));
   show_bin((pointer)&start[i+0],sizeof(unsigned char));

}

int main(){

   float f = 4.0f;
   float g = 5.0f;
   float h;

    
   show_float((pointer)&f);
   printf("\n");
   show_bytes((pointer)&f,sizeof(float));
   
//   h = 4.0;

   
   show_float((pointer)&g);
   printf("\n");
   show_bytes((pointer)&g,sizeof(float));
   
/*   
   show_float((pointer)&h);
   printf("\n");
   show_bytes((pointer)&h,sizeof(float));
*/

}

