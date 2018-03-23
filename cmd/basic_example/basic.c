#include <stdio.h>

int main(int argc, char* argv[]) {
	char flag[] = "flag{basic_flag}";
	char* fp;
	char buf[0x100];
	fp = flag;

	fgets(buf, 0x100, stdin);

	printf(buf);

	return 0;
}
