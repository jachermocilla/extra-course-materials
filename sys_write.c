//gcc -S -masm=intel -O0 sys_write.c

#include <unistd.h>

unsigned char data[]={'H','e','l','l','o',',','w','o','r','l','d','!',10};

int write_131(){
   write(1,(char *)data, 13);
}

int main(){
   write_131();
}
