/**
 * $export OMP_NUM_THREADS=2
 * $gcc -fopenmp -osum_omp.exe sum_omp.c
 *
 */



#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int sum; /* this data is shared by the thread(s) */

int main(int argc, char *argv[])
{
   if (argc != 2) {
      fprintf(stderr,"usage: a.out <integer value> \n");
      return -1;
   }

   if (atoi(argv[1]) < 0) {
      fprintf(stderr,"%d must be >= 0 \n",atoi(argv[1]));
      return -1;
   }

   #pragma omp parallel
   {   
   int i, upper = atoi(argv[1]);
   sum = 0;
   for (i = 1; i <= upper; i++)
      sum += i;
      sleep(10);
   }

   printf("sum = %d \n",sum);
}


