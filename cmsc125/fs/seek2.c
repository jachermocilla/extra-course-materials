#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

struct _data{
   char name[10];
   int age;
};

struct _data students[10];

int main(){
   FILE *fp=fopen("students.db","rb");
   struct _data student; 
 
   fseek(fp,7*sizeof(struct _data),SEEK_SET);
   fread(&student,sizeof(struct _data),1,fp);
   fclose(fp);
   printf("%s %d\n",student.name,student.age);
   return 0;
}


