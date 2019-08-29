#include <stdio.h>
#include <stdlib.h>

int main(){
   printf("%d\n", (0 == 0U));
   printf("%d\n", (-1 < 0));
   printf("%d\n", (-1 < 0U));
   printf("%d\n", (2147483647 > (-2147483647-1)));
   printf("%d\n", (2147483647U > (-2147483647-1)));
   printf("%d\n", (2147483647 > ((int)2147483647U)));
   printf("%d\n", (-1 > -2));
   printf("%d\n", ((unsigned) -1 > -2 ));
   printf("%d\n", ( 0 <= 0U ));

}
