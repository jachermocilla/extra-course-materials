/*
 * Use 10 threads
 *
 * $export OMP_NUM_THREADS=10
 * $gcc -fopenmp -o simple_omp.exe simple_omp.c
 *
 */

#include <omp.h>
#include <stdio.h>
int main(int argc, char *argv[])
{
   /* sequential code */
   #pragma omp parallel
   {
      printf("I am a parallel region.\n");
   }
   /* sequential code */
}

