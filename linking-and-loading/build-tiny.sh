#!/bin/bash

nasm -f elf tiny.asm
gcc -Wall -m32 -s -nostartfiles tiny.o -o tiny.elf
