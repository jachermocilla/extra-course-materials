
all: child.elf parent.elf parent2.elf orphan.elf zombie.elf

clean:
	rm child.elf parent.elf parent2.elf orphan.elf zombie.elf

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

