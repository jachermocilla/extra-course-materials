#include<pthread.h>
#include<stdio.h>
#include<unistd.h>

#define i 0
#define j 1
#define niter 10

#define true 1
#define false 0

void *thread_i(void *);
void *thread_j(void *);


int flag[2];
int turn=0;
int shared_data=100;

int main()
{
   pthread_t tid_i,tid_j;

   printf("Initial shared_data= %d\n",shared_data);


   pthread_create(&tid_i,NULL,thread_i,NULL);
   pthread_create(&tid_j,NULL,thread_j,NULL);
   pthread_join(tid_i,NULL);
   pthread_join(tid_j,NULL);
}

void *thread_i(void *param){
   int c=0;

   while(c<niter)
   {
      flag[i]=true;
      turn=j;
      while(flag[j] && turn==j)
         ;
      
      shared_data+=100;
      
      flag[i]=false;
      
      printf("Process i: shared_data= %d\n",shared_data);

      c++;
      sleep(2);   
   }
}

void *thread_j(void *param)
{
   int c=0;

   while(c<niter)
   {
      flag[j]=true;
      turn=i;
      while(flag[i] && turn==i)
         ;
      
      shared_data-=75;
      
      flag[j]=false;
      
      printf("Preocess j: shared_data= %d\n",shared_data);

      c++;
      sleep(3);

   }
}
