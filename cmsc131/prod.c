//Code used for blog entry on GDB
//https://jachermocilla.blogspot.com/2019/09/introduction-to-debugging-c-programs.html

#include <stdio.h>
#include <stdlib.h>

int mul(int x, int y){
   int prod;
   int i;

   prod=0;
   for (i=0;i<y;i++){
      prod=prod+x;
   }
   return prod;
}

int main(){
   int a=4;
   int b=3;

   printf("The product of %d and %d is %d\n",a,b,mul(a,b));
   
   return 0;

}
