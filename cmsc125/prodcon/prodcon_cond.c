#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>

//
//solution to prod-con problem using a counter to be able to use  
//all slots in the buffer. it uses condition variables.
//-jach
//

#define BUFFER_SIZE 5
#define NITER 15

//Function prototypes
void *producer();
void *consumer();

//Shared by the producer and consumer thread
int buffer[BUFFER_SIZE];
int in = 0;    //next free position 
int out = 0;   //first full position

//Counter to hold the total number of elements
//We use this in order to fill all slots.
int counter=0;


pthread_mutex_t mutex;                    //for mutual exclusion
pthread_cond_t item_available;            //event that an item is available
pthread_cond_t slot_available;            //event that a slot is available

int main(){
   pthread_t prod_thread, con_thread;
   srand(time(NULL));

   pthread_mutex_init(&mutex,NULL);                 //initialize mutex and condition variables
   pthread_cond_init(&item_available, NULL);

   pthread_create (&prod_thread, NULL, producer, NULL);
   pthread_create (&con_thread, NULL, consumer, NULL);

   pthread_join(prod_thread, NULL);
   pthread_join(con_thread, NULL);

   printf("Counter: %d\n",counter);

   return 0;
}



void *producer(){
   int next_produced; 
   //while (1){
   for (int i=0; i< NITER; i++) { 
      /* produce an item in next produced */ 
      next_produced=rand() % 100 + 1;


      pthread_mutex_lock(&mutex);

      while (counter==BUFFER_SIZE)                 
         pthread_cond_wait(&slot_available, &mutex);      //wait for an event that a slot is available

      buffer[in] = next_produced;
      printf("Producer produced [%d].(Placed in index:in=%d,out=%d)\n",next_produced,in,out);     
      in = (in + 1) % BUFFER_SIZE; 
      counter++;

      pthread_cond_signal(&item_available);           //signal that an item is available
      pthread_mutex_unlock(&mutex);
   }
}

void *consumer(){
   int next_consumed; 
   //while(1){ 
   for (int i=0; i< NITER; i++) { 

      pthread_mutex_lock(&mutex);

      while (counter==0) 
         pthread_cond_wait(&item_available, &mutex);      //wait for an event that an item is available
         

      next_consumed = buffer[out]; 
      printf("\t\tConsumer consumed [%d].(in=%d,Consumed from index: out=%d)\n",next_consumed,in,out);     
      out = (out + 1) % BUFFER_SIZE;
      counter--;
      
      pthread_cond_signal(&slot_available);           //signal that a slot is available
      pthread_mutex_unlock(&mutex);

   } 
}
