#!/bin/bash  
ts=$(date +%s%N)  
./non-vector.elf
echo "Non-vector: $((($(date +%s%N) - $ts)/1000000)) ms" > times.txt 

ts=$(date +%s%N)  
./vector-avx2.elf
echo "Vector (avx2): $((($(date +%s%N) - $ts)/1000000)) ms" >> times.txt

cat times.txt
