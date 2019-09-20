#include <stdio.h>

int logical(int x, int y){
   int t1 = x ^ y;
   int t2 = t1 >> 17;
   int mask = (1<<13) - 7;
   int rval = t2  & mask;
   return rval;
}


int main(){
  logical(7,10); 
}
