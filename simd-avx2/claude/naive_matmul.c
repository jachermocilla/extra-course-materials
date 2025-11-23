#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

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
    m->data = (elem_t*)malloc(rows * cols * sizeof(elem_t));
    return m;
}

void matrix_free(Matrix *m) {
    if (m) {
        free(m->data);
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

void matrix_multiply_naive(Matrix *a, Matrix *b, Matrix *c) {
    if (a->cols != b->rows) {
        fprintf(stderr, "Incompatible matrix dimensions\n");
        return;
    }

    memset(c->data, 0, c->rows * c->cols * sizeof(elem_t));

    for (int i = 0; i < a->rows; i++) {
        for (int j = 0; j < b->cols; j++) {
            elem_t sum = 0;
            for (int k = 0; k < a->cols; k++) {
                sum += *matrix_at(a, i, k) * *matrix_at(b, k, j);
            }
            *matrix_at(c, i, j) = sum;
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
    matrix_multiply_naive(a, b, c);
    clock_gettime(CLOCK_MONOTONIC, &end);

    double elapsed = (end.tv_sec - start.tv_sec) +
                     (end.tv_nsec - start.tv_nsec) / 1e9;

    printf("%.6f\n", elapsed);

    matrix_free(a);
    matrix_free(b);
    matrix_free(c);

    return 0;
}
