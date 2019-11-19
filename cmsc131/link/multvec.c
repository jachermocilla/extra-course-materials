void multvec(int *x, int *y, int *z, int n){
   int i;
   
   for (i=0; i < n; i++)
      x[i] = x[i] * y[i];
}
