#!/bin/bash

set -e

echo "=========================================="
echo "Matrix Multiplication Benchmark"
echo "=========================================="
echo ""

# Compile naive version
echo "Compiling naive version..."
gcc -O3 -march=native -o naive_matmul naive_matmul.c -lm -lrt
echo "✓ Naive version compiled"
echo ""

# Compile AVX2 version
echo "Compiling AVX2 version..."
gcc -O3 -march=native -mavx2 -o avx2_matmul avx2_matmul.c -lm -lrt
echo "✓ AVX2 version compiled"
echo ""

# Test sizes
sizes=(256 512 1024)

# Run benchmarks
echo "Running benchmarks..."
echo ""
printf "%-12s %-20s %-20s %-15s\n" "Matrix Size" "Naive (seconds)" "AVX2 (seconds)" "Speedup"
printf "%-12s %-20s %-20s %-15s\n" "-----------" "---------------" "---------------" "--------"

for size in "${sizes[@]}"; do
    # Run naive version 3 times and take average
    naive_total=0
    for i in {1..3}; do
        result=$(./naive_matmul "$size")
        naive_total=$(echo "$naive_total + $result" | bc -l)
    done
    naive_avg=$(echo "scale=6; $naive_total / 3" | bc -l)

    # Run AVX2 version 3 times and take average
    avx2_total=0
    for i in {1..3}; do
        result=$(./avx2_matmul "$size")
        avx2_total=$(echo "$avx2_total + $result" | bc -l)
    done
    avx2_avg=$(echo "scale=6; $avx2_total / 3" | bc -l)

    # Calculate speedup
    speedup=$(echo "scale=2; $naive_avg / $avx2_avg" | bc -l)

    printf "%-12s %-20s %-20s %-15s\n" "$size x $size" "$naive_avg" "$avx2_avg" "${speedup}x"
done

echo ""
echo "=========================================="
echo "Benchmark complete!"
echo "=========================================="
