#include <stdio.h>
#include <stdlib.h>

float sum_elements(float a[], unsigned length){
   int i;
   float result=0;
   
   for (i=0;i<=length-1;i++)
      result += a[i];
   return result;
}


int main(){
   float data[5]={1.0,2.0,3.0,4.0,5.0};
   
   printf("sum = %f\n",sum_elements(data,5));
   //printf("sum = %f\n",sum_elements(data,0));

}
