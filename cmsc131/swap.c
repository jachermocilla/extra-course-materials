#include <stdio.h>
#include <stdlib.h>

void swap(int *xp, int *yp){
   int t0 = *xp;
   int t1 = *yp;
   *xp = t1;
   *yp = t0;
}

int main(){
   int a=123;
   int b=456;
   
   printf("a=%d,b=%d\n",a,b);
   swap(&a,&b);
   printf("a=%d,b=%d\n",a,b);


}
