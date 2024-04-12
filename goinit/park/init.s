.global _start

// Constants for syscalls
.equ SYS_OPENAT, 56
.equ SYS_WRITE, 64
.equ SYS_CLOSE, 57
.equ SYS_EXIT, 93
.equ O_WRONLY, 0x0001
.equ O_CREAT, 0x0040
.equ O_TRUNC, 0x0200
.equ O_APPEND, 0x0400
.equ AT_FDCWD, -100

.data
    // File path
    filepath:   .asciz "/dev/kmsg"
    // Text message to write
    message:    .asciz "Hello, kernel log!\n"

.text
_start:
    // Open /dev/kmsg
    mov x0, AT_FDCWD              // working directory (AT_FDCWD for current directory)
    ldr x1, =filepath             // address of file path
    mov x2, O_WRONLY              // flags for open (write-only)
    mov x3, 0                     // mode (ignored when flags include O_CREAT)
    mov x8, SYS_OPENAT            // syscall number for openat
    svc #0                        // syscall

    // Check if file descriptor is valid
    cmp x0, #0
    blt open_failed               // branch if open failed

    // Write message to /dev/kmsg
    mov x1, x0                    // file descriptor for write
    ldr x2, =message              // address of message
    ldr x3, =14                   // length of message
    mov x8, SYS_WRITE             // syscall number for write
    svc #0                        // syscall

    // Close /dev/kmsg
    mov x0, x1                    // file descriptor to close
    mov x8, SYS_CLOSE             // syscall number for close
    svc #0                        // syscall

    // Exit program
    mov x0, #0                    // exit status
    mov x8, SYS_EXIT              // syscall number for exit
    svc #0                        // syscall

open_failed:
    // Print error message and exit
    ldr x0, #4                    // address of error message
    mov x8, SYS_EXIT              // syscall number for exit
    svc #0                        // syscall
