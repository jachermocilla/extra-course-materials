#include <windows.h> 
#include <stdio.h> 
#include <tchar.h>
#define BUFSIZE 512
#define pipename "\\\\.\\pipe\\LogPipe"
int main(int argc, TCHAR *argv[]) 
{ 
    while (1)
    {
        HANDLE pipe = CreateNamedPipe(pipename, PIPE_ACCESS_INBOUND | PIPE_ACCESS_OUTBOUND , PIPE_WAIT, 1, 1024, 1024, 120 * 1000, NULL);

        if (pipe == INVALID_HANDLE_VALUE)
        {
           printf("Error: %d", GetLastError());
        }

        char data[1024];
        DWORD numRead;

        ConnectNamedPipe(pipe, NULL);

        ReadFile(pipe, data, 1024, &numRead, NULL);

        if (numRead > 0)
            printf("%s\n", data);
            
        CloseHandle(pipe);
    }
  return 0; 
}
