#include <stdio.h>
#include <stdlib.h>

char big_array[1<<24]; // 16MB
char huge_array[1<<28]; //256MB

int beyond;
char *p1, *p2, *p3, *p4;

int useless() {return 0;}

int main(){
   p1 = malloc(1<<28);
   p2 = malloc(1<<8);
   p3 = malloc(1<<28);
   p4 = malloc(1<<8);

   printf("Done.");
}

