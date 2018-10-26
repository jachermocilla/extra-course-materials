#!/bin/bash
#encrypt.sh, chmod 755

START=$(date +%s%N | cut -b1-13)
for ((i=0; i<5000; i++))
do
   echo "enc/dec"
   #insert openssl commands to encrypt or decrypt here
   #in the loop's body
done
END=$(date +%s%N | cut -b1-13)

DIFF=$(($END - $START))
TRIES=5000
AVG=$(($DIFF / $TRIES))
DECIMAL=$(($DIFF % $TRIES))
echo "AVG TIME: $AVG.$DECIMAL ms"
