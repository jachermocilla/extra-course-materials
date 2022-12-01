#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>


struct _data{
   char name[10];
   int age;
};

struct _data students[10];

int main(){
   FILE *fp=fopen("students.db","rb");
   struct _data student; 

   int rc=fork(); 

   if (rc == 0){
      fseek(fp, sizeof(struct _data),SEEK_SET);
      printf("child: offset %ld\n",ftell(fp));
      sleep(30);
   }else if ( rc > 0){
      wait(NULL);
      printf("parent: offset %ld\n",ftell(fp));
   } 
   fclose(fp);
   return 0;
}


