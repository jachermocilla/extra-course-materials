#include <stdio.h>
#include <stdlib.h>

void swapa(long *xp, long *yp){
   volatile long loc[2];
   loc[0] = *xp;
   loc[1] = *yp;
   *xp = loc[1];
   *yp = loc[0];
}

int main(){
   long a=123;
   long b=456;
   
   printf("a=%ld,b=%ld\n",a,b);
   swapa(&a,&b);
   printf("a=%ld,b=%ld\n",a,b);


}
