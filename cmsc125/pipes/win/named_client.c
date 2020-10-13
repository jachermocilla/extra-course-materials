#include <windows.h> 
#include <stdio.h> 
#include <tchar.h>
#define MAX_BUF 1024
#define pipename "\\\\.\\pipe\\LogPipe"
int main()
{
    HANDLE pipe = CreateFile(pipename, GENERIC_READ | GENERIC_WRITE, 0, NULL, OPEN_EXISTING, 0, NULL);

    if (pipe == INVALID_HANDLE_VALUE)
    {
        printf("Error: %d", GetLastError());
    }

    char message [] = "Hi";

    DWORD numWritten;
    
    WriteFile(pipe, message, 3, &numWritten, NULL); 

    return 0;
}
