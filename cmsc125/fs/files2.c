#include <stdio.h>
#include <stdlib.h>
int main()
{
   FILE *fp;char buf[64];
   fp=fopen("boo.txt","r");
   fgets(buf,64,fp);
   fclose(fp);
   return 0;
}

