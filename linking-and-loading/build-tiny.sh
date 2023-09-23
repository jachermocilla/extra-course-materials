#!/bin/bash

nasm -f elf tiny.asm
gcc -Wall -m32 -s -nostdlib -nostartfiles tiny.o -o tiny.elf
