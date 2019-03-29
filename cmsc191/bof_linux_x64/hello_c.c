// hello_c.c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
char name[10];			// global variable

void update(char new_name[10],char new_location[10]){
   char location[10];   	// local variable 
   strcpy(location,new_location);
   strcpy(name,new_name);
}
int main(){
  static short int z; 	// uninitialized data (goes to .bss)
  char *school;         	// variable on the heap
  z = 0xAABB;
  update("JACH","COLLEGE");
  school = (char *)malloc(10);
  strcpy(school,"UPLB");
  free(school);
  return 0;
}
