#include <stdio.h>
#include <stdlib.h>

void incrk(int *ip, int k){
   *ip += k;
}

int add3(int x){
   int localx = x;
   incrk(&localx,3);
   return localx;
}

int main(){
   add3(3); 
}
