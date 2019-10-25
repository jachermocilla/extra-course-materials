#include <stdio.h>

struct S1{
   char c;           //
   int i[2];
   double v;
}*p;

int main(){
   struct S1 s;
   printf("sizeof(int): %ld bytes\n",sizeof(int));
   printf("sizeof(char): %ld bytes\n",sizeof(char));
   printf("sizeof(double): %ld bytes\n",sizeof(double));
   printf("sizeof(stuct S1): %ld bytes\n",sizeof(struct S1));
}
