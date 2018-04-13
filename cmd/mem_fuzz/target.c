#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#define FILENAME "./data.dat"
#define BUFSZ 0x100

void print_welcome() {
	printf("Welcome\n");
}

int parse_file() {
	char* buf;
	
	buf = (char*)malloc(BUFSZ);

	read(0, buf, BUFSZ);

	// start processing the file
	if (*(buf + (BUFSZ/2)) != 'J') {
		return -1;
	}

	if (*buf != 'Q') {
		return -1;
	}

	if (*(buf + 8) != 'P') {
		return -1;
	}

	if (*(buf + (BUFSZ-1)) != 'U') {
		return -1;
	}

	printf("Success!\n");
	printf("You did it!\n");
	return 0;
}

int main(int argc, char* argv[]) {
	print_welcome();

	if (parse_file()) {
		printf("parse error!\n");
		return 1;
	}
	return 0;
}
