/**
    >cl bof.c /DYNAMICBASE:NO /GS- /Gs- /link /FIXED
    >editbin bof.exe /NXCOMPAT:NO
*/

int main(int argc, char *argv[]){
	char buf[20];
/*	
	if (argc < 2){
		puts("Usage: bof.exe <name>\n");
		exit(1);
	}
	strcpy(buf,argv[1]);
*/
	printf("Enter name: ");
	gets(buf);
	printf("Hello %s!\n",buf);
}