#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <regex.h>
#include <unistd.h>
#include <time.h>

void main() {
	char *patterns[] = {
		"j784s4-evm login:",
		"root@j784s4-evm:~#"
	};
	char *actions[] = {
		"root",
		"ls /"
	};
	setvbuf(stdout, NULL, _IONBF, 0);
	int pos = 0;
	size_t num_patterns = sizeof(patterns) / sizeof(patterns[0]);

	while (pos < num_patterns) {
		char input_str[4096];
		if (fgets(input_str, sizeof(input_str), stdin) == NULL) {
			break;
		}
		input_str[strcspn(input_str, "\n")] = '\0';

		regex_t regex;
		if (regcomp(&regex, patterns[pos], REG_EXTENDED) != 0) {
			fprintf(stderr, "Error compiling regex pattern\n");
			exit(EXIT_FAILURE);
		}

		if (regexec(&regex, input_str, 0, NULL, 0) == 0) {
			fprintf(stderr, "found %d\n", pos);
			printf("%s\n", actions[pos]);
			pos++;
		}
		regfree(&regex);
	}

	fprintf(stderr, "script terminated\n");
	sleep(2);
}
