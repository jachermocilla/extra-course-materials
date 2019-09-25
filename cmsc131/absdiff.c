int absdiff(int x, int y){
   int result;
   if (x > y)
      result = x - y;
   else
      result = y - x;
   return result;
}


int main(){
   absdiff(4,5);
}
