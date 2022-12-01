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
   FILE *fp2=fopen("students.db","rb");
   struct _data student; 
   struct _data student2; 
 
   fseek(fp,7*sizeof(struct _data),SEEK_SET);
   fread(&student,sizeof(struct _data),1,fp);

   fseek(fp2,5*sizeof(struct _data),SEEK_SET);
   fread(&student2,sizeof(struct _data),1,fp2);

   sleep(50);
   fclose(fp);
   fclose(fp2);

   printf("%s %d\n",student.name,student.age);
   printf("%s %d\n",student2.name,student2.age);
   return 0;
}


