#include <stdio.h>
#include <stdlib.h>

   
int course1 = 15213;
int course2 = 18243;

void swap(int *xp, int *yp){
   int t0 = *xp;
   int t1 = *yp;
   *xp = t1;
   *yp = t0;
}

void call_swap(){
   swap(&course1, &course2);
}

int main(){
   call_swap();
}
