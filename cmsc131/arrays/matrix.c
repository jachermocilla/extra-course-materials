#define N 16
typedef int fix_matrix[N][N];

int fix_ele(fix_matrix a, int i, int j){
   return a[i][j];
}

#define IDX(n, i, j) ((i)*(n)+(j))
int vec_ele(int n, int *a, int i, int j){
   return a[IDX(n,i,j)];

}

int var_ele(int n, int a[n][n], int i, int j){
   return a[i][j];
}


void fix_column(fix_matrix a, int j, int *dest){
   int i;
   for (i=0;i<N;i++){
      dest[i] = a[i][j];
   }
}


int main(){}

