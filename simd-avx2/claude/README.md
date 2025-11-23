## Usage
```bash
make benchmark
make distclean
```


### How using AVX2 works
```bash
matrix_multiply_avx2(A, B, C)
   A: n × k          B: k × m          C: n × m
          ┌───────────────────────────────┐
          │  For each row i in A (0..n-1)   │
          └───────────────────────────────┘
                    │
                    ▼
       ┌────────────────────────────────┐
       │  For j = 0..m-1 step 8 (vectorized) │
       └────────────────────────────────┘
                    │
                    ▼
             sum = [0,0,0,0,0,0,0,0]   ← __m256i zero vector
                    │
                    ▼
       ┌──────────────────────────────────────┐
       │  For each p = 0..k-1 (inner dimension)│
       └──────────────────────────────────────┘
              │                │
      a_elem = A[i][p]         │
              │                │
              ▼                ▼
      a_vec = [a_elem ×8]   b_vec = B[p][j .. j+7]   (8 consecutive cols)
              │                │
              ├───────×───────┤
              ▼                ▼
                prod = a_vec * b_vec       ← _mm256_mullo_epi32
                         │
                         ▼
                   sum += prod             ← _mm256_add_epi32
                         │
                    (loop k times)
                         │
                         ▼
            After inner loop: sum = 8 final values for C[i][j .. j+7]
                         │
                         ▼
            _mm256_storeu_si256 → C[i][j .. j+7]   (write 8 elements at once)

       ──────────────────────────────────────────────────────────────
       Remaining columns (j % 8 ≠ 0):
            Scalar loop: normal C[i][j] += A[i][p] * B[p][j] for p=0..k-1
```
