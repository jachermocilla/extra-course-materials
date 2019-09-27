#include <stdio.h>

#define WSIZE 8*sizeof(int)

int pcount_for(unsigned x){
   int i;
   int result = 0;
   for (i = 0; i < WSIZE; i++) {
      unsigned mask = 1 << i;
      result += (x & mask) != 0 ;
   }
   return result;
}

int main(){
   printf("%d\n",pcount_for(1));
   printf("%d\n",pcount_for(3));
}
