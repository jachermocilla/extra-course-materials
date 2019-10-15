#include <stdio.h>
#include <stdlib.h>

void swap(long *xp, long *yp){
   volatile long loc[2];
   loc[0] = *xp;
   loc[1] = *yp;
   *xp = loc[1];
   *yp = loc[0];
}

void swap_ele(long a[], int i){
   swap(&a[i], &a[i+1]);
}

long sum = 0;
void swap_ele_sum(long a[], int i){
   swap(&a[i],&a[i+1]);
   sum += (a[i]*a[i+1]);
}



int main(){
   long a=123;
   long b=456;
   
   printf("a=%ld,b=%ld\n",a,b);
   swap(&a,&b);
   printf("a=%ld,b=%ld\n",a,b);


}
