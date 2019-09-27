#include <stdio.h>

int pcount_while(int x){
   int result = 0;
   while (x){
      result += x & 0x1;
      x >>= 1;
   }
   return result;
}

int main(){
   printf("%d\n",pcount_while(1));
   printf("%d\n",pcount_while(3));
}
