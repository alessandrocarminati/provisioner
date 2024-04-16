#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <regex.h>
#include <errno.h>
#include <setjmp.h>
#include <signal.h>
#include <unistd.h>
#include <termios.h>

#define NUM_PATTERNS 9
#define TIMEOUT 5
#define MAX_LINE_LENGTH 1024

struct termios orig_termios;

struct line {
	int pos;
	char buf[MAX_LINE_LENGTH + 1];
};

void reset_terminal_mode()
{
	tcsetattr(0, TCSANOW, &orig_termios);
}

void set_conio_terminal_mode()
{
	struct termios new_termios;

	tcgetattr(0, &orig_termios);
	memcpy(&new_termios, &orig_termios, sizeof(new_termios));
	atexit(reset_terminal_mode);
	cfmakeraw(&new_termios);
	tcsetattr(0, TCSANOW, &new_termios);
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


int getchar_timeout(char *c, int timeout_seconds) {
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
		return read(STDIN_FILENO, c, 1);
	} else {
		return 0;
	}
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

void update_line(struct line *l, char c) {
	if ((c == '\b') || (c == 0x7f)) { // Handle backspace
		if (l->pos > 0) {
			memmove(&l->buf[l->pos - 1], &l->buf[l->pos], strlen(&l->buf[l->pos]) + 1);
			l->pos--;
		}
	} else if ((c == '\r')||(c == '\n')) { // Handle carriage return/newline
		memset(l->buf, 0, MAX_LINE_LENGTH);
		l->pos = 0;
	} else if (c >= 32 && c <= 126) { // Handle printable characters
		if (l->pos < MAX_LINE_LENGTH) {
			l->buf[l->pos] = c;
			l->pos++;
		}
	} else if (c == 0x03) { // Handle Ctrl+C
		exit(0); // Exit program
	}

}

int main() {
	char *patterns[NUM_PATTERNS] = {
		"^=> ",
		"^=> ",
		"^=> ",
		"^=> ",
		"^=> ",
		"^=> ",
		"^=> ",
		"^=> ",
		"^buildroot login:"
	};
	char *actions[NUM_PATTERNS] = {
		"echo dummy",
		"echo dummy",
		"dhcp",
		"setenv serverip 10.26.28.75",
		"tftpboot 0x82000000 J784S4XEVM.flasher.img",
		"tftpboot 0x84000000 k3-j784s4-evm.dtb",
		"setenv bootargs rootwait root=/dev/mmcblk1p3",
		"booti 0x82000000 - 0x84000000",
		"root"
	};

	char c;
	struct line *current_line;
	int ret, i, pos=0;

	set_conio_terminal_mode();

	// unbuffered stdout
	setvbuf(stdout, NULL, _IONBF, 0);
	current_line = (struct line *)malloc(sizeof(struct line));
	if (current_line == NULL) {
		fprintf(stderr, "Memory allocation error\n");
		exit(EXIT_FAILURE);
	}

	while (pos <NUM_PATTERNS) {
		ret = getchar_timeout(&c, TIMEOUT);
		if (ret<0) {
			fprintf(stderr, "select error\n");
			continue;
		}
		if (!ret) {
			fprintf(stderr, "timeout\n");
			printf("\n");
			fflush(stdout);
			continue;
		}
		update_line(current_line, c);
		if (find_occurrence(current_line->buf, patterns, pos)){
			fprintf(stderr, "'%s' found, print %s\n", patterns[pos], actions[pos]);
			sleep(1);
			printf("%s\n", actions[pos]);
			pos++;
			update_line(current_line, '\n');
		}
	}

	return 0;
}
