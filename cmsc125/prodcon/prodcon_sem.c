#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>
#include <semaphore.h>

//
// implementation of solution to prod-con using 
// semaphores.
// -jach
//
//




//The number of elements in the buffer
#define BUFFER_SIZE     5             

//Number of producer threads
#define NPROD           1              

//Number of consumer threads
#define NCON            1              

//Number of iterations in the producer
#define PRODITER        20              

//Number of iterations in the consumer
#define CONITER         20              

//Function prototypes
void *producer(void *);
void *consumer(void *);

//Shared data by the producer and consumer thread
//------------------------------------------------
int buffer[BUFFER_SIZE];
int in = 0;    //next free position 
int out = 0;   //first full position

sem_t mutex;   //semaphore to lock buffer
sem_t full;    //counts the number of filled slots 
sem_t empty;   //counts the number of empty slots
//------------------------------------------------

int main(){
   //Array if thread ids for producer
   pthread_t prod_thread[NPROD];

   //Array of thread ids for consumer
   pthread_t con_thread[NCON];
   int i,j;

   //Placeholder of parameters passed to the thread, the id
   int *prod_id,*con_id;

   //Initialize the random number generator
   srand(time(NULL));

   sem_init(&mutex,0,1);            //we lock the buffer
   sem_init(&full,0,0);             //no items yet
   sem_init(&empty,0,BUFFER_SIZE);  //all buffers are empty 

   for (i=0;i<NPROD;i++){
      prod_id=(int *)malloc(sizeof(int));
      *prod_id=i;
      pthread_create (&prod_thread[0], NULL, producer, prod_id);
   }
   
   for (j=0;j<NCON;j++){
      con_id=(int *)malloc(sizeof(int));
      *con_id=j;
      pthread_create (&con_thread[0], NULL, consumer, con_id);
   }


   for (i=0;i<NPROD;i++){
      pthread_join(prod_thread[i], NULL);
   }   

   for (j=0;j<NCON;j++){
      pthread_join(con_thread[j], NULL);
   }

   return 0;
}


void *producer(void *i){
   int next_produced;
   int prod_id=*((int *)i);
   int full_val,empty_val;
   //while (1){
   for (int i=0; i< PRODITER; i++) { 
      /* produce an item in next produced by generating a random number*/ 
      next_produced=rand() % 100 + 1;

      sem_wait(&empty);       //wait for an empty slot
      sem_wait(&mutex);       //wait for lock to access shared data:buffer, in

      //critical section
      buffer[in] = next_produced; 


      printf("Producer %d produced [%d].(Placed in index: in=%d,out=%d,),",
               prod_id, next_produced,in,out);     
      in = (in + 1) % BUFFER_SIZE; 

      sem_post(&mutex);       //release the lock on the shared data
      sem_post(&full);        //signal that an item is available

      //get the value of the semaphores
      sem_getvalue(&empty,&empty_val);
      sem_getvalue(&full,&full_val);
      printf("semaphores: empty=%d,full=%d\n",empty_val,full_val);
   }
}

void *consumer(void *i){
   int next_consumed; 
   int con_id=*((int *)i);
   int full_val,empty_val;
   //while(1){ 
   for (int i=0; i< CONITER; i++) { 
      sem_wait(&full);        //wait for an item to be available
      sem_wait(&mutex);       //wait for the lock to access shared data: buffer, out

      //critical section
      next_consumed = buffer[out]; 
      
      //get the value of the semaphores
      sem_getvalue(&empty,&empty_val);
      sem_getvalue(&full,&full_val);
      
      printf("\t\tConsumer %d consumed [%d].(in=%d,Consumed from: out=%d),",
                  con_id,next_consumed,in,out);     
      out = (out + 1) % BUFFER_SIZE;

      sem_post(&mutex);       //release the lock on the shared data
      sem_post(&empty);       //signal that a new item has been consumed
      
      //get the value of the semaphores
      sem_getvalue(&empty,&empty_val);
      sem_getvalue(&full,&full_val);
      printf("semaphores: empty=%d,full=%d\n",empty_val,full_val);

      /* consume the item in next consumed */ 
   } 
}