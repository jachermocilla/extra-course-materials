//https://gist.github.com/jbwyme/9479813
#include<stdio.h>
#include<stdlib.h>
#include<pthread.h>
#include<unistd.h>

pthread_key_t key;

struct args {
	char *msg;
};

void print_msg(char *msg) {
	int *tl = (int *)pthread_getspecific(key);
	printf("msg: %s, tl: %d\n", msg, *tl);
}

void *exec_in_thread(void *param) {
   struct args *args = (struct args *)param;

	int *tl = malloc(sizeof(int));
	*tl = 5;
	pthread_setspecific(key, tl);
	print_msg(args->msg);
	sleep(2);
	*tl = 4;
	print_msg(args->msg);
	pthread_setspecific(key, NULL);
	free(tl);
	pthread_exit(NULL);
}

int main() {
	int i = 0, num_threads = 10;
	pthread_t threads[num_threads];
	struct args *my_args = malloc(sizeof(struct args));
	my_args->msg = "some message...";
	pthread_key_create(&key, NULL);
	for(i = 0; i<num_threads; i++) {
		pthread_create(&threads[i], NULL, exec_in_thread, my_args);
	}
	for(i = 0; i<num_threads; i++) {
		pthread_join(threads[i], NULL);
	}
	return 0;
}
