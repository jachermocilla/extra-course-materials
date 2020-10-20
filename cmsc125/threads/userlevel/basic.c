/**
 * References:
 * https://brennan.io/2020/05/24/userspace-cooperative-multitasking/
*/
#include <stdio.h>
#include <setjmp.h>

jmp_buf saved_location;
int main(int argc, char **argv)
{
    if (setjmp(saved_location) == 0) {
        printf("We have successfully set up our jump buffer!\n");
    } else {
        printf("We jumped!\n");
        return 0;
    }

    printf("Getting ready to jump!\n");
    longjmp(saved_location, 1);
    printf("This will never execute...\n");
    return 0;
}


