#!/usr/bin/env python3
import sys
import re
import time

def main():
    patterns = [
        "j784s4-evm login:",
        "root@j784s4-evm:~#"
    ]
    actions = [
        "root",
        "ls /"
    ]
    pos = 0

    while pos < len(patterns):
        try:
            input_str = sys.stdin.readline().strip()

            if not input_str:
                continue

            if re.search(patterns[pos], input_str):
                print("found", pos, file=sys.stderr)
                print(actions[pos], flush=True)
                pos=pos+1
        except Exception as e:
            print("An exception occurred:", e, file=sys.stderr)

    print("script terminated", file=sys.stderr)
    time.sleep(2)

if __name__ == '__main__':
    main()
