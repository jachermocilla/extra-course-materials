/**
    >cl hello.c -Zi /DYNAMICBASE:NO /GS- /Gs- /link /FIXED /DEBUG
    >editbin hello.exe /NXCOMPAT:NO
*/

int main(int argc, char *argv[]){
	printf("Hello!\n");
}