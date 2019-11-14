struct rec{
   int a[3];
   int i;
   struct rec *n;
};
   
struct rec my_rec;
struct rec *p_my_rec=&my_rec;

void set_i(struct rec *r, int val){
   r->i = val;
}

int *get_ap(struct rec *r, int  idx){
   return &r->a[idx];
}

void set_val(struct rec *r, int val){
   while (r){
      int i = r->i;
      r->a[i] = val;
      r= r->n;
   }
}

int main(){

   my_rec.i=10;
   set_i(p_my_rec,7); 


}
