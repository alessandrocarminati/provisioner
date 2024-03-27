#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <string.h>

int main(){
	const char *message = "Starting things#######\n";
	size_t message_len = strlen(message);

	int fd = open("/dev/kmsg", O_WRONLY);
	write(fd, message, message_len);
	close(fd);
	int onefd = open("/dev/console", O_RDONLY, 0);
	dup2(onefd, 0); // stdin
	int twofd = open("/dev/console", O_RDWR, 0);
	dup2(twofd, 1); // stdout
	dup2(twofd, 2); // stderr

	if (onefd > 2) close(onefd);
	if (twofd > 2) close(twofd);

	printf("hello==========================================>\n");
	getchar();

}
