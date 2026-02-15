# goinit

## Overview

**goinit** is a task-specific Linux init designed for provisioning and flashing workflows.

It replaces a traditional initramfs runtime with a minimal execution environment dedicated to board provisioning.

Typical responsibilities:

-   Flash root filesystems
-   Write kernel images
-   Install DTBs
-   Update U-Boot environment
-   Execute scripted actions

It is tightly integrated with Provisioner as the on-board agent.

## Design Philosophy

-   Minimal userspace
-   Deterministic execution
-   Kernel-embedded runtime
-   Remote controllability
-   Provisioning-only scope

## Build

Build:
```
make
```

Output artifact:
```
bin/rootfs.cpio
```

## Kernel Integration

Embed via initramfs:
```
CONFIG_INITRAMFS_SOURCE="<path>/rootfs.cpio"
CONFIG_INITRAMFS_ROOT_UID=0
CONFIG_INITRAMFS_ROOT_GID=0
```
The resulting kernel becomes a self-contained provisioning image bootable by U-Boot.

## Boot Arguments

Controlled via `pr.*` kernel parameters.

| Argument      | Description                  |
|---------------|------------------------------|
| pr.ifname     | Interface to bring up (DHCP) |
| pr.syslogIP   | Syslog server                |
| pr.action     | Action to execute            |
| pr.actionArgX | Action arguments             |
| pr.debuglevel | Verbosity                    |
| pr.reboot     | Reboot after completion      | 
| pr.apiPort    | HTTP API port                |

## Control API

Available when networking is up.

### System Status

```
GET /api/stat
```

Returns CPU, memory, storage, network.

### Read Device
```
GET /api/read?device=/dev/sda
```

### Write Device

```
POST /api/write?device=/dev/sda
```

### Reboot
```
POST /api/reboot
```

## Management IP Reporting

After DHCP:
```
PROVISIONER_MGMT_IF=<iface>
PROVISIONER_MGMT_IP=<ip>
```

Provisioner parses this via serial or syslog to transition control from console to network API.

## Role in Provisioner Ecosystem

goinit acts as the **on-board provisioning agent** responsible for:

-   Flash execution
-   Artifact ingestion
-   Device exposure
-   Runtime reporting

Together they form a full provisioning pipeline from power-on to flashed system.
