;; nasm -f elf64 -o hello.o hello.asm
;; ld -o hello.elf hello.o


BITS 64
section .data
    msg db "Hello, World!", 0xA  ; String to print, 0xA is newline character
    len equ $ - msg             ; Length of the string

section .text
    global _start               ; Entry point for the linker

_start:
    ; System call for write (sys_write = 1)
    mov rax, 1                  ; System call number for sys_write
    mov rdi, 1                  ; File descriptor for standard output (stdout)
    mov rsi, msg                ; Address of the string to print
    mov rdx, len                ; Length of the string
    syscall                     ; Invoke the system call

    ; System call for exit (sys_exit = 60)
    mov rax, 60                 ; System call number for sys_exit
    xor rdi, rdi                ; Exit code 0
    syscall                     ; Invoke the system call
