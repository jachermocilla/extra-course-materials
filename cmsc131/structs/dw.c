union {
   unsigned char c[8];
   unsigned short s[4];
   unsigned int i[2];
   unsigned long l[1];
}dw;

int main(){
   dw.c[0]=0x8;
   dw.c[0]=0x9;
   dw.c[0]=0xA;
   dw.c[0]=0xB;
   dw.c[0]=0xC;
   dw.c[0]=0xD;
   dw.c[0]=0xE;
   dw.c[0]=0xF;

}
