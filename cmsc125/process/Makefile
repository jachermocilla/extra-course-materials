
all: child.elf parent.elf parent2.elf orphan.elf zombie.elf cpu-bound.elf io-bound.elf

clean:
	rm child.elf parent.elf parent2.elf orphan.elf zombie.elf cpu-bound.elf io-bound.elf

child.elf: child.c
	gcc -o child.elf child.c -Wall

parent.elf: parent.c
	gcc -o parent.elf parent.c -Wall

parent2.elf: parent2.c
	gcc -o parent2.elf parent2.c -Wall

orphan.elf: orphan.c
	gcc -o orphan.elf orphan.c -Wall

zombie.elf: zombie.c
	gcc -o zombie.elf zombie.c -Wall


cpu-bound.elf: cpu-bound.c
	gcc -o cpu-bound.elf cpu-bound.c -Wall

io-bound.elf: io-bound.c
	gcc -o io-bound.elf io-bound.c -Wall

