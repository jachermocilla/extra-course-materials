#define _GNU_SOURCE     //to be able to use pthread_setname_np()

//implementation of solution to prod-con from chapter 3, dino book
//not all slots are used 
//-jach
//


#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>
#include <unistd.h>

#define B_TRUE 1
#define B_FALSE 0


//Function prototypes
void *t1_func();
void *t2_func();

int flag = B_FALSE;
int x = 0;

int main(){
   pthread_t t1, t2;
   srand(time(NULL));

   pthread_create (&t1, NULL, t1_func, NULL);
   pthread_create (&t2, NULL, t2_func, NULL);

   pthread_setname_np(t1, "Thread 1"); 
   pthread_setname_np(t2, "Thread 2"); 

   pthread_join(t1, NULL);
   pthread_join(t2, NULL);

   return 0;
}



void *t1_func(){
	while (!flag)
		;
		//asm volatile ("mfence" : : : "memory");
	sleep(1+rand() %3);
	printf("x=%d\n",x);
}

void *t2_func(){
	x = 100;
	//asm volatile ("mfence" : : : "memory");
	sleep(1+rand()%3);
	flag = B_TRUE;
}
