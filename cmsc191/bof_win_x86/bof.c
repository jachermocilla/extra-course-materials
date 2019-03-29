/**
    >cl bof.c -Zi /DYNAMICBASE:NO /GS- /Gs- /link /FIXED /DEBUG
    >editbin bof.exe /NXCOMPAT:NO
*/
int main(int argc, char *argv[]){
	char buf[20];
	printf("Enter name: ");
	gets(buf);
	printf("Hello %s!\n",buf);
}