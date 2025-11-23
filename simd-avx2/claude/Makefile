CC := gcc
CFLAGS := -O3 -march=native -Wall -Wextra
LDFLAGS := -lm -lrt

# Target executables
TARGETS := naive_matmul avx2_matmul
SOURCES := naive_matmul.c avx2_matmul.c
OBJECTS := $(SOURCES:.c=.o)

# Default target
.PHONY: all
all: $(TARGETS)

# Naive implementation
naive_matmul: naive_matmul.o
	$(CC) $(CFLAGS) -o $@ $^ $(LDFLAGS)

naive_matmul.o: naive_matmul.c
	$(CC) $(CFLAGS) -c -o $@ $<

# AVX2 implementation
avx2_matmul: avx2_matmul.o
	$(CC) $(CFLAGS) -mavx2 -o $@ $^ $(LDFLAGS)

avx2_matmul.o: avx2_matmul.c
	$(CC) $(CFLAGS) -mavx2 -c -o $@ $<

# Run benchmark
.PHONY: benchmark
benchmark: all
	@chmod +x benchmark.sh
	@./benchmark.sh

# Clean up object files
.PHONY: clean
clean:
	rm -f $(OBJECTS)

# Remove everything including executables
.PHONY: distclean
distclean: clean
	rm -f $(TARGETS)

# Rebuild everything
.PHONY: rebuild
rebuild: distclean all

# Help target
.PHONY: help
help:
	@echo "Matrix Multiplication Benchmark - Makefile targets:"
	@echo ""
	@echo "  make all        - Build both naive and AVX2 implementations (default)"
	@echo "  make benchmark  - Build and run the benchmark comparison"
	@echo "  make clean      - Remove object files"
	@echo "  make distclean  - Remove object files and executables"
	@echo "  make rebuild    - Clean and rebuild everything"
	@echo "  make help       - Show this help message"

.PHONY: info
info:
	@echo "Compiler: $(CC)"
	@echo "CFLAGS: $(CFLAGS)"
	@echo "LDFLAGS: $(LDFLAGS)"
	@echo "Targets: $(TARGETS)"
