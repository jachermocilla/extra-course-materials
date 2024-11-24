#include <stdio.h>
#include <stdlib.h>

#define N 8

float array_a[8] = { 125.45, 293.82, 411.67, 87.39, 254.12, 375.98, 189.74, 468.23 };
float array_b[8] = { 72.19, 348.56, 129.47, 402.38, 215.74, 89.63, 367.45, 478.92 };
float array_c[8] = { 563.24, 120.89, 487.56, 98.31, 372.67, 248.94, 413.52, 59.84 };

void multiply_and_add(const float* a, const float* b, const float* c, float* d) {  
   int i;
   for(int i=0; i<N; i++) {
      d[i] = a[i] * b[i];
      d[i] = d[i] + c[i];
   }   
}

int main(){
   float result[N];
   multiply_and_add(array_a,array_b,array_c,result);
   for(int i=0; i<N; i++) {
      printf("[%d]: %.2f\n",i,result[i]);
   }   
}
