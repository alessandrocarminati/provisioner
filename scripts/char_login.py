#!/usr/bin/env python3
import sys
import tty
import time
import threading

def main():
    buffer = ""
    running = True
    lock = threading.Lock()
    search_strings = ["=> ", "=> ", "=> ", "=> ", "=> ", "=> ", "=> ", "buildroot login:"]
    action = ["echo dummy", "dhcp", "setenv serverip 10.26.28.75", "tftpboot 0x82000000 J784S4XEVM.flasher.img", "tftpboot 0x84000000 k3-j784s4-evm.dtb", "setenv bootargs rootwait root=/dev/mmcblk1p3", "booti 0x82000000 - 0x84000000", "root"]
    last_found_index = 0

    def get_char():
        byte = sys.stdin.read(1)
        return byte


    def input_loop():
        nonlocal buffer, running
        while running:
            print(f"current buffer= {buffer}",file=sys.stderr)
            c = get_char()
            if c == '\n' or c == '\r':
                with lock:
                    buffer = ""
            elif c =='\x7f':
                with lock:
                    buffer = buffer[:-1]
            elif c =='\x03':
                exit()
            else:
                with lock:
                    buffer += c

    input_thread = threading.Thread(target=input_loop)
    input_thread.start()

    while last_found_index<len(search_strings):
        if buffer.find(search_strings[last_found_index]) != -1:
            print(action[last_found_index], flush=True)
            print(f"{search_strings[last_found_index]} found, print {action[last_found_index]}",file=sys.stderr)
            buffer = ""
            last_found_index=last_found_index+1
        time.sleep(0.05)

    running = False
    input_thread.join()

if __name__ == "__main__":
    main()
