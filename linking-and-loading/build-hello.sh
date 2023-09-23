#!/bin/bash

nasm -f elf64 hello.asm
ld hello.o -o hello.elf
./hello.elf ; echo $?
