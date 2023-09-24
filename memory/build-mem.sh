#!/bin/bash

nasm -f bin mem.asm -o mem.bin
qemu-system-i386 -m 128M -fda mem.bin -boot a
