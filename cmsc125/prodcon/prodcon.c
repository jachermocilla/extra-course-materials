#define _GNU_SOURCE     //to be able to use pthread_setname_np()

//implementation of solution to prod-con from chapter 3, dino book
//not all slots are used 
//-jach
//


#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>


#define BUFFER_SIZE 5
#define NITER 10


//Function prototypes
void *producer();
void *consumer();

//Shared by the producer and consumer thread
int buffer[BUFFER_SIZE];
int in = 0;    //next free position 
int out = 0;   //first full position


int main(){
   pthread_t prod_thread, con_thread;
   srand(time(NULL));

   pthread_create (&prod_thread, NULL, producer, NULL);
   pthread_create (&con_thread, NULL, consumer, NULL);

   pthread_setname_np(prod_thread, "ProducerThread"); 
   pthread_setname_np(con_thread, "ConsumerThread"); 

   pthread_join(prod_thread, NULL);
   pthread_join(con_thread, NULL);

   return 0;
}



void *producer(){
   int next_produced; 
   //while (1){
   for (int i=0; i< NITER; i++) { 
      /* produce an item in next produced */ 
      next_produced=rand() % 100 + 1;

      //Do nothing until a slot is available 
      while (((in + 1) % BUFFER_SIZE) == out) 
         ; 

      buffer[in] = next_produced; 
      printf("Producer produced [%d].(Placed in index:in=%d,out=%d)\n",next_produced,in,out);     
      in = (in + 1) % BUFFER_SIZE; 
   }
}

void *consumer(){
   int next_consumed; 
   //while(1){ 
   for (int i=0; i< NITER; i++) { 
      
      //Do nothing if the buffer is empty
      while (in == out) 
         ; 

      next_consumed = buffer[out]; 
      printf("\t\tConsumer consumed [%d].(in=%d,Consumed from index: out=%d)\n",next_consumed,in,out);     
      out = (out + 1) % BUFFER_SIZE;

      /* consume the item in next consumed */ 
   } 
}