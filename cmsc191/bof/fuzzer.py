#!/usr/bin/python
#fuzzer.py
#n must be adjusted so that the return 
#address, as shown in gdb, is not overwritten by As (0x41). 

n = 100
print 'A' * n
