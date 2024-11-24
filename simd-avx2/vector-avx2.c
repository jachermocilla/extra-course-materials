#include <stdio.h>
#include <stdlib.h>
#include <immintrin.h>
/*
 $gcc -o vector-avx2.elf vector-avx2.c -mavx2
*/



int main(){
   //These should be inside a function 
   // __m256 is the datatype for 256-bit vector containing 8 floats
   // _set_ps - "packed single" precision (floats)
   __m256 vector_a = _mm256_set_ps(125.45, 293.82, 411.67, 87.39, 254.12, 375.98, 189.74, 468.23);
   __m256 vector_b = _mm256_set_ps(72.19, 348.56, 129.47, 402.38, 215.74, 89.63, 367.45, 478.92);
   __m256 vector_c = _mm256_set_ps(563.24, 120.89, 487.56, 98.31, 372.67, 248.94, 413.52, 59.84);

   __m256 vector_d = _mm256_mul_ps(vector_a,vector_b);
   vector_d = _mm256_add_ps(vector_d,vector_c);

   float *result = (float *) &vector_d;

   for(int i=0; i<8; i++) {
      printf("[%d]: %.2f\n",i,result[i]);
   }   
}


