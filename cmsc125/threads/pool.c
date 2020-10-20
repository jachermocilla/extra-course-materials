//https://gist.github.com/bodokaiser/5689562
#include "queue.h"
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pthread.h>

#define THREADS 3

/**
 * Task queue.
 */
QUEUE queue;

/**
 * Type of a calc work task.
 */
typedef struct {
    int a;
    int b;
    int type;
    QUEUE node;
} work_t;

/**
 * Our threads.
 */
pthread_t threads[THREADS];

/**
 * Our thread condition variable.
 */
pthread_cond_t cond;

/**
 * Our thread mutex lock.
 */
pthread_mutex_t mutex;

/* function headers */
void * worker();
void submit_work(int a, int b, int type);

/**
 * Should execute the submited work tasks through thread pool.
 */
int main(void) {
    QUEUE_INIT(&queue);

    pthread_cond_init(&cond, NULL);
    pthread_mutex_init(&mutex, NULL);

    /* 3 x 3 = 9 */
    submit_work(3, 3, 1);
    /* 4 - 3 = 1 */
    submit_work(4, 3, 2);
    /* 7 * 8 = 56 */
    submit_work(7, 8, 3);
    /* 30 / 6 = 5 */
    submit_work(30, 6, 4);

    /* start all threads */    
    for (int i = 0; i < THREADS; i++)
        pthread_create(&threads[i], NULL, worker, NULL);

    /* wait all threads to finish */
    for (int i = 0; i < THREADS; i++)
        pthread_join(threads[i], NULL);

    pthread_mutex_destroy(&mutex);
    pthread_cond_destroy(&cond);

    return EXIT_SUCCESS;
}

/**
 * Adds a calculation task to queue.
 */
void submit_work(int a, int b, int type) {
    /* dynamically allocate a work task */
    work_t * work = malloc(sizeof(work_t));

    work->a = a;
    work->b = b;
    work->type = type;

    /* lock the queue to avoid thread access */
    pthread_mutex_lock(&mutex);

    /* add work task to work queue */
    QUEUE_INIT(&work->node);
    QUEUE_INSERT_TAIL(&queue, &work->node);

    /* free the lock */
    pthread_mutex_unlock(&mutex);

    /* signal a thread that it should check for new work */
    pthread_cond_signal(&cond);
}

/**
 * Worker thread.
 *
 * Looks for new tasks to execute.
 */
void * worker() { 
    QUEUE * q;

    int result;

    work_t * work;

    for (;;) {
        pthread_mutex_lock(&mutex);
        
        while (QUEUE_EMPTY(&queue)) {
            pthread_cond_wait(&cond, &mutex);
        }

        q = QUEUE_HEAD(&queue);
        
        QUEUE_REMOVE(q);

        pthread_mutex_unlock(&mutex);
        
        work = QUEUE_DATA(q, work_t, node);

        switch (work->type) {
            case 1:
                result = work->a + work->b;
                printf("%d + %d = %d\n", work->a, work->b, result);
                break;
            case 2:
                result = work->a - work->b;
                printf("%d - %d = %d\n", work->a, work->b, result);
                break;
            case 3:
                result = work->a * work->b;
                printf("%d * %d = %d\n", work->a, work->b, result);
                break;
            case 4:
                result = work->a / work->b;
                printf("%d / %d = %d\n", work->a, work->b, result);
                break;
        }

        free(work);
    }

    pthread_exit(NULL);
}
