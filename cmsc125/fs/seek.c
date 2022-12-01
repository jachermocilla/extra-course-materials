#include <stdio.h>
#include <stdlib.h>
#include <string.h>

struct _data{
   char name[10];
   int age;
};

struct _data students[10];

int main(){
   FILE *fp=fopen("students.db","wb");

   for (int i=0;i<10;i++){
      sprintf(students[i].name,"%d",i);
      students[i].age=i;
   }
   
   fwrite(&students,sizeof(struct _data),10,fp);
   fclose(fp);
   return 0;
   

}


