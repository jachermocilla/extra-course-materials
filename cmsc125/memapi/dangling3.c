#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct _node{
   int data;
   struct _node *next;
}*node_ptr;

void print_list(node_ptr head){
   node_ptr tmp;
   tmp=head;
   while (tmp!=NULL){
      printf("%d\n",tmp->data);
      tmp = tmp->next;
   }
}

int main(){
   node_ptr a,b,c;

   a=(node_ptr)malloc(sizeof(struct _node));
   a->data=1;

   b=a->next=(node_ptr)malloc(sizeof(struct _node));
   b->data=2;

   c=a->next->next=(node_ptr)malloc(sizeof(struct _node));
   c->data=3;
   c->next=NULL;

   print_list(a);
   
   free(b);

   print_list(a);
}


