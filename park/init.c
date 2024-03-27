#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <string.h>
#include <errno.h>
#include <dirent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/sysmacros.h>
#define CONDEF "./con"

void find_recursive(const char *path) {
    DIR *dir;
    struct dirent *entry;
    struct stat statbuf;

    if ((dir = opendir(path)) == NULL) {
        perror("opendir");
        return;
    }

    while ((entry = readdir(dir)) != NULL) {
        char full_path[1024];
        sprintf(full_path, "%s/%s", path, entry->d_name);

        if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0)
            continue;

        printf("%s\n", full_path);

        if (stat(full_path, &statbuf) == -1) {
            perror("stat");
            continue;
        }

        if (S_ISDIR(statbuf.st_mode))
            find_recursive(full_path);
    }

    closedir(dir);
}

int main(){
	const char *message = "Starting things#######\n";
	size_t message_len = strlen(message);

	const char *device_name = CONDEF;
	int major_number = 5;
	int minor_number = 1;
	dev_t dev = makedev(major_number, minor_number);

	if (mknod(device_name, S_IFCHR | 0666, dev) < 0) {
		exit(0x55);
	}
	char *s = getenv("CONSOLE");
	if (!s) s = getenv("console");
	if (!s) s = CONDEF;
	int fd = open(s, O_RDWR | O_NONBLOCK | O_NOCTTY);
	if (fd >= 0) {
		dup2(fd, 0);
		dup2(fd, 1);
		dup2(fd, 2);
	} else {
//		exit(s==CONDEF?0x11:0x22);
		exit(errno);
	}

	write(1, message, message_len);
	printf("stocazzo %s\n", s);
	find_recursive("/");

	return s==CONDEF?0xff:0x7f;
}
