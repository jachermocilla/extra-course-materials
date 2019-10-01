long switch_eg(long x, long y, long z){
   long w = 1;

   switch(x) {
      case 1:
         w = y*z;
         break;
      case 2:
         w = y/z;
         /* Fall Through */
      case 3:
         w += z;
         break;
      case 5:
      case 6:
         w -= z;
         break;
      default:
         w = 2;
   }
   return w;
}

int main(){
   switch_eg(1,2,3);

}
