#include <stdio.h>
#include <stdlib.h>

void swap(long *xp, long *yp){
   long t0 = *xp;
   long t1 = *yp;
   *xp = t1;
   *yp = t0;
}

int main(){
   long a=123;
   long b=456;
   
   printf("a=%ld,b=%ld\n",a,b);
   swap(&a,&b);
   printf("a=%ld,b=%ld\n",a,b);


}
