/**
    >cl simple.c -Zi /DYNAMICBASE:NO /GS- /Gs- /link /FIXED /DEBUG
    >editbin simple.exe /NXCOMPAT:NO
*/
#include <stdio.h>
#include <string.h>

void greeter(char *name){
        char buffer[128];
        strcpy(buffer,name);
        printf("%s",buffer);
}

int main(int argc, char **argv){
        greeter(argv[1]);
        return 0;
}
