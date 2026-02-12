# Provisioner Project

## Description

The Provisioner project aims to develop a comprehensive provisioning sharing
application for common u-boot based boards. This application is designed to
be self-contained, providing a seamless experience for efficiently managing
and provisioning resources. It acts as a single "sidekick" per board (one
board, one provisioner).

![image info](./imgs/provisioner.drawio.png)

---

## Key Functionality

### Serial connection management
The application takes ownership of the serial connection to the board's
console and extends access externally through SSH (tunnel and monitor
sessions).

### Control plane via SSH
A control plane accessible through SSH serves as a CLI for board management:
power control (PDU via SNMP, Tasmota, or Beaker), user enable/disable for
the tunnel, script execution, serial logging, and file transfer over serial.

### TFTP and HTTP services
- **TFTP**: Kernel image provisioning; supports local files and HTTP proxy
  (e.g. `http___...` filenames).
- **HTTP**: Serves binary artifacts (rootfs, kernel, device trees) from a
  configurable directory.

### Google Calendar integration
Optional integration for access control and reservations (calendar poller
and ACL).

### Kernel image and goinit
A kernel image embeds an initram with scripts for flashing; a companion
**goinit** component can act as a flasher agent on the board.

## Recent changes and features

- **Monitor CLI:** currently supports:
    - Tab **autocomplete** for commands;
    - **Ctrl-C** cancels the current line;
    - **Ctrl-W** kills the entire line.
- **Send file over serial:** `send_serial <file> <mode> [dest_path]` with 
  modes **plain** (base64), **gzip** (base64 + gzip), 
  **xmodem_unix** (e.g. `rx` on the board), **xmodem_uboot** (`loadx`). 
  Provisioner issues the remote command and runs the transfer; if the receiver
  does not respond (e.g. no NAK/CRC for XMODEM), an error is returned.
  Dependencies (stty, base64, gunzip, rx) are the user's responsibility.

## Scripting: assm vs exec_scr

The tool supports two ways to automate interaction with the serial console:

### Native scripts (assm)
- **assm** is a small assembly-like language (see `scripts/`).
  Scripts wait for suffixes (e.g. "login : ", "assword: "), then send
  predefined strings.
 Execution is character-oriented (one byte per `fetch`).
- **Recommendation:** Prefer **assm for expect-style flows** (login, wait for
  prompt, send response).
  It is designed for that pattern and avoids the need to write 
  character-oriented code in a general-purpose language.

### External scripts (exec_scr)
- **exec_scr** runs an external executable (any language) and connects its 
  stdin/stdout to the serial stream. You choose **line** or **char** mode when
  invoking:
    - **line**: Provisioner sends data to the script only when a line is complete
      (after Enter). The script can use normal line-based I/O (`readline()`, 
      `input()`, etc.).
    - **char**: Provisioner sends every byte as it arrives. The script must read
      byte-by-byte, buffer, and match patterns (expect-style); this is not a common
      pattern in most languages.
