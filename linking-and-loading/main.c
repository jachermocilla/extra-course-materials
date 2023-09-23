extern void a(char *);

int main(int argc, char **argv){
   static char string[]="Hello, world!\n";
   a(string);
}
