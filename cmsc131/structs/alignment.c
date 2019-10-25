#include <stdio.h>

struct S1{
   char c;           //
   int i[2];
   double v;
}*p;

//assumes little endian
void printBits(size_t const size, void const * const ptr)
{
    unsigned char *b = (unsigned char*) ptr;
    unsigned char byte;
    int i, j;

    for (i=size-1;i>=0;i--)
    {
        for (j=7;j>=0;j--)
        {
            byte = (b[i] >> j) & 1;
            printf("%u", byte);
        }
    }
    puts("");
}


int main(){
   struct S1 s;
   printf("sizeof(int): %ld bytes\n",sizeof(int));
   printf("sizeof(char): %ld bytes\n",sizeof(char));
   printf("sizeof(double): %ld bytes\n",sizeof(double));


   printf("sizeof(s): %ld bytes\n",sizeof(s));
   printf("addressof(s): %p bytes\n",&s);
   printf("sizeof(s.c): %ld bytes\n",sizeof(s.c));
   printf("addressof(s.c): %p bytes\n",&s.c);
   printf("sizeof(s.i): %ld bytes\n",sizeof(s.i));
   printf("addressof(s.i): %p bytes\n",&s.i);
   printf("sizeof(s.v): %ld bytes\n",sizeof(s.v));
   printf("addressof(s.v): %p bytes\n",&s.v);



//   printBits(sizeof(double *),&s);
//   printBits(sizeof(char *),&s.c);
//   printBits(sizeof(int *),&s.i);
//   printBits(sizeof(double *),&s.v);

}
