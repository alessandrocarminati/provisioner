#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <regex.h>
#include <errno.h>
#include <setjmp.h>
#include <signal.h>
#include <unistd.h>

#define BUFFER_SIZE 4096
#define NUM_PATTERNS 2
#define TIMEOUT 5

int resizebuf(char **buf) {
	char *newline_pos = strrchr(*buf, '\n');
	if (newline_pos == NULL) {
		return strlen(*buf);
	}
	char *new_buf = (char *)malloc(BUFFER_SIZE * sizeof(char));
	for (int i=0; i<BUFFER_SIZE; i++) *(new_buf+i)=0;
	if (new_buf == NULL) {
		fprintf(stderr, "Memory allocation error\n");
		exit(EXIT_FAILURE);
	}
	strcpy(new_buf, newline_pos + 1);
	free(*buf);
	*buf = new_buf;
	return 0;
}

int find_occurrence(char *buf, char *patterns[], int index) {
	regex_t regex;
	if (regcomp(&regex, patterns[index], REG_EXTENDED | REG_NOSUB) != 0) {
		fprintf(stderr, "Error compiling regex pattern %d\n", index+1);
		exit(EXIT_FAILURE);
	}

	int found = regexec(&regex, buf, 0, NULL, 0) == 0;

	regfree(&regex);

	return found;
}

int read_timeout(char *buffer, int size, int timeout_seconds) {
	fd_set fds;
	struct timeval tv;
	int retval;

	FD_ZERO(&fds);
	FD_SET(STDIN_FILENO, &fds);

	tv.tv_sec = timeout_seconds;
	tv.tv_usec = 0;

	retval = select(STDIN_FILENO + 1, &fds, NULL, NULL, &tv);

	if (retval == -1) {
		perror("select()");
		return -1;
	} else if (retval) {
		return read(STDIN_FILENO, buffer, size);
	} else {
		return 0;
	}
}


int main() {
	char *patterns[NUM_PATTERNS] = {
		"j784s4-evm login:",
		"root@j784s4-evm:~#"
	};
	char *actions[NUM_PATTERNS] = {
		"root",
		"ls /"
	};
	int ret, bufpos=0, pos=0;

	setvbuf(stdout, NULL, _IONBF, 0);
	char *buffer = (char *)malloc(BUFFER_SIZE * sizeof(char));
	for (int i=0; i<BUFFER_SIZE; i++) *(buffer+i)=0;
	if (buffer == NULL) {
		fprintf(stderr, "Memory allocation error\n");
		exit(EXIT_FAILURE);
	}

	while (pos <NUM_PATTERNS) {
		ret = read_timeout(buffer+bufpos, BUFFER_SIZE, TIMEOUT);
		if (!ret) {
			fprintf(stderr, "timeout\n");
			printf("\n");
			fflush(stdout); 
		}
		if (find_occurrence(buffer, patterns, pos)){
			fprintf(stderr, "found, print %s\n", actions[pos]);
			printf("%s\n", actions[pos]);
			pos++;
		}
		bufpos = resizebuf(&buffer);
	}

	free(buffer);
	return 0;
}
