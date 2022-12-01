#include <stdio.h>
#include <stdlib.h>
int main()
{
   FILE *fp;
   fp=fopen("boo.txt","w");
   fputs("I <3 cmsc125!",fp);
   fclose(fp);
   return 0;
}

