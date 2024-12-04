#!/bin/bash  

gcc -o vector-avx2.elf vector-avx2.c -mavx2
gcc -o non-vector.elf non-vector.c 

ts=$(date +%s%N)  
./non-vector.elf
echo "Non-vector: $((($(date +%s%N) - $ts)/1000000)) ms" > times.txt 

ts=$(date +%s%N)  
./vector-avx2.elf
echo "Vector (avx2): $((($(date +%s%N) - $ts)/1000000)) ms" >> times.txt

cat times.txt
