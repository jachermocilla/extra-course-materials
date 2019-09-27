#include <stdio.h>

int pcount_do(int x){
   int result = 0;
   do {
      result += x & 0x1;
      x >>= 1;
   }while (x);
   return result;
}

int main(){
   printf("%d\n",pcount_do(1));
   printf("%d\n",pcount_do(3));
}
