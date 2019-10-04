#include <stdio.h>

int pcount_r(unsigned x){
   if (x==0)
      return 0;
   else
      return (x & 1) + pcount_r(x >> 1);
}

int main(){
   printf("%d\n",pcount_r(3));
}
