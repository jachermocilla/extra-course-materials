#define ZLEN 5
#define ISIZE sizeof(int)
typedef int zip_dig[ZLEN];

zip_dig cmu = { 1, 5, 2, 1, 3 };
zip_dig mit = { 0, 2, 1, 3, 9 };
zip_dig ucb = { 9, 4, 7, 2, 0 };


#define PCOUNT 4
zip_dig pgh[PCOUNT] = 
   {  {1,5,2,0,6},
      {1,5,2,1,3},
      {1,5,2,1,7},
      {1,5,2,2,1} };


#define UCOUNT 3
int *univ[UCOUNT] = {mit, cmu, ucb};

int get_univ_digit(int index, int dig){
   return univ[index][dig];
}


int *get_pgh_zip(int index){
   return pgh[index];
}


int get_pgh_digit(int index, int dig){
   return pgh[index][dig];
}

int get_digit(zip_dig z, int dig){
   return z[dig];
}

void zincr(zip_dig z){
   int i;
   for (i = 0; i < ZLEN; i++)
      z[i]++;
}

void zincr_p(zip_dig z){
   int *zend = z + ZLEN;
   do {
      (*z)++;
      z++;
   } while (z != zend);
}

void zincr_v(zip_dig z){
   void *vz=z;
   int i=0;
   do{
      (*((int *) (vz+i)))++;
      i += ISIZE;
   } while (i != ISIZE*ZLEN);
}



int main(){}
