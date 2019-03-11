#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>

//
//solution to prod-con problem using a counter to be able to use  
//all slots in the buffer. Uses compare and swap
//-jach
//

#define BUFFER_SIZE 5
#define NITER 100

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


unsigned long lock=0;

static inline unsigned long
compare_and_swap(volatile unsigned long* value, unsigned long expected, unsigned long new_value)
{
    unsigned long prev;
    asm volatile("lock;"
                 "cmpxchgq %1, %2;"
                 : "=a"(prev)
                 : "q"(new_value), "m"(*value), "a"(expected)
                 : "memory");
    return prev;
}


int main(){
   pthread_t prod_thread, con_thread;
   srand(time(NULL));

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

      //Do nothing when the buffer is full
      while (counter==BUFFER_SIZE) 
         ; 

      buffer[in] = next_produced;
      //printf("Producer produced [%d].(Placed in index:in=%d,out=%d)\n",next_produced,in,out);     
      in = (in + 1) % BUFFER_SIZE; 

      while (compare_and_swap(&lock, 0, 1) != 0)
         ; /* do nothing */

      //critical section
      counter++;

      lock=0;
   }
}

void *consumer(){
   int next_consumed; 
   //while(1){ 
   for (int i=0; i< NITER; i++) { 

      //Do nothing until an item is available
      while (counter==0) 
         ;

      next_consumed = buffer[out]; 
      //printf("\t\tConsumer consumed [%d].(in=%d,Consumed from index: out=%d)\n",next_consumed,in,out);     
      out = (out + 1) % BUFFER_SIZE;

      while (compare_and_swap(&lock, 0, 1) != 0)
         ; /* do nothing */

      //critical section
      counter--;
      
      lock=0;

   } 
}
