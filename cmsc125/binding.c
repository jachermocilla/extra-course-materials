/**
 * jachermocilla@gmail.com
 *
 * $ gcc -fno-common -c -o binding.o binding.c
 * $ gcc -fno-commin -o binding.exe binding.c
 *
*/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>


//global variable of primitive type, initialized to non-zero
char data_global_initialized_nonzero[32]="JACH_IN_DATA";

//global variable of primitive type, initialized to zero
int bss_global_initialized_zero=0;

//structures
struct _bss_struct{
   char field1;
   short field2;
}bss_global_struct;

//global variable of string
char bss_global[32];

//global variable, 2d array
int bss_global_matrix[30][30];

//a global pointer variable
char *bss_global_pointer;

//a function with a parameter
int text_func(char stack_parameter[32]){
   printf("stack_parameter: %p\n",&stack_parameter);
   return 0;
}

//a function, program entry point
int main(){
   char stack_local[32]="JACH_IN_STACK_LOCAL";         
   int *stack_local_pointer;      //local pointer variable


   strcpy(bss_global,"JACH_IN_BSS");

   bss_global_pointer=(char *)malloc(sizeof(32));     //allocate memory
   strcpy(bss_global_pointer,"JACH_IN_HEAP");

   stack_local_pointer=(int *)malloc(sizeof(int));     //allocate memory
   *stack_local_pointer=40;

   //display the logical address at runtime
   printf("text_main: %p\n",(void *)main);
   printf("text_func: %p\n",(void *)text_func);
   printf("data_global_initialized_nonzero: %p\n",&data_global_initialized_nonzero);
   printf("bss_global_initialized_zero: %p\n",&bss_global_initialized_zero);
   printf("bss_global_struct: %p\n",&bss_global_struct);
   printf("bss_global: %p\n",&bss_global);
   printf("bss_global_matrix: %p\n",&bss_global_matrix);
   printf("stack_local: %p\n",&stack_local);
   text_func("JACH_IN_STACK_PARAM");
   printf("bss_global_pointer: %p\n",&bss_global_pointer);
   printf("pointed to by bss_global_pointer (in heap): %p\n",(void *)bss_global_pointer);
   printf("stack_local_pointer: %p\n",&stack_local_pointer);
   printf("pointed to by stack_local_pointer (in heap): %p\n",(void *)stack_local_pointer);

   return 0;
}
