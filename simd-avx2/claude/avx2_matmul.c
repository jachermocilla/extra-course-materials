#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <immintrin.h>

typedef int32_t elem_t;

typedef struct {
    int rows;
    int cols;
    elem_t *data;
} Matrix;

Matrix* matrix_create(int rows, int cols) {
    Matrix *m = (Matrix*)malloc(sizeof(Matrix));
    m->rows = rows;
    m->cols = cols;
    m->data = (elem_t*)_mm_malloc(rows * cols * sizeof(elem_t), 32);
    return m;
}

void matrix_free(Matrix *m) {
    if (m) {
        _mm_free(m->data);
        free(m);
    }
}

void matrix_random_fill(Matrix *m, int seed) {
    srand(seed);
    for (int i = 0; i < m->rows * m->cols; i++) {
        m->data[i] = (elem_t)(rand() % 100);
    }
}

elem_t* matrix_at(Matrix *m, int row, int col) {
    return &m->data[row * m->cols + col];
}

void matrix_multiply_avx2(Matrix *a, Matrix *b, Matrix *c) {
    if (a->cols != b->rows) {
        fprintf(stderr, "Incompatible matrix dimensions\n");
        return;
    }

    memset(c->data, 0, c->rows * c->cols * sizeof(elem_t));

    int n = a->rows;
    int m = b->cols;
    int k = a->cols;

    for (int i = 0; i < n; i++) {
        for (int j = 0; j < m; j += 8) {
            // Create a 256-bit vector filled with zeros (8 x int32 = 0)
            __m256i sum = _mm256_setzero_si256();

            for (int p = 0; p < k; p++) {
                elem_t a_elem = *matrix_at(a, i, p);

                // Replicate one int32 value into all 8 lanes of a 256-bit vector
                // Result: [a_elem, a_elem, a_elem, a_elem, a_elem, a_elem, a_elem, a_elem]
                __m256i a_vec = _mm256_set1_epi32(a_elem);

                // Load 8 consecutive int32 values from matrix B (unaligned load)
                // From address: &b->data[p * m + j] loads 8 values
                __m256i b_vec = _mm256_loadu_si256((__m256i*)matrix_at(b, p, j));

                // Multiply corresponding elements: a_vec[i] * b_vec[i] for all 8 lanes
                // Result: 8 multiplied values in one vector
                __m256i prod = _mm256_mullo_epi32(a_vec, b_vec);

                // Add the 8 products to the accumulator
                // sum[i] += prod[i] for all 8 lanes
                sum = _mm256_add_epi32(sum, prod);
            }

            // Store the 8 accumulated results back to matrix C (unaligned store)
            // Writes 8 int32 values to: &c->data[i * m + j]
            _mm256_storeu_si256((__m256i*)matrix_at(c, i, j), sum);
        }

        // Handle remaining columns that don't fit in 8-element chunks
        // If m is not divisible by 8, process leftover columns with scalar code
        for (int j = (m / 8) * 8; j < m; j++) {
            elem_t result = 0;
            for (int p = 0; p < k; p++) {
                result += *matrix_at(a, i, p) * *matrix_at(b, p, j);
            }
            *matrix_at(c, i, j) = result;
        }
    }
}

int main(int argc, char *argv[]) {
    if (argc < 2) {
        fprintf(stderr, "Usage: %s <matrix_size>\n", argv[0]);
        return 1;
    }

    int size = atoi(argv[1]);
    Matrix *a = matrix_create(size, size);
    Matrix *b = matrix_create(size, size);
    Matrix *c = matrix_create(size, size);

    matrix_random_fill(a, 42);
    matrix_random_fill(b, 43);

    struct timespec start, end;
    clock_gettime(CLOCK_MONOTONIC, &start);
    matrix_multiply_avx2(a, b, c);
    clock_gettime(CLOCK_MONOTONIC, &end);

    double elapsed = (end.tv_sec - start.tv_sec) +
                     (end.tv_nsec - start.tv_nsec) / 1e9;

    printf("%.6f\n", elapsed);

    matrix_free(a);
    matrix_free(b);
    matrix_free(c);

    return 0;
}
