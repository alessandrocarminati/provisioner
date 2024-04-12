# Provisioner Script Samples
Welcome to the script directory of the Provisioner project!
This directory contains sample scripts that illustrate how to automate tasks
using the Provisioner tool.

## Purpose
The scripts provided here serve as examples to demonstrate how to write
scripts for automating tasks within the Provisioner tool.
These scripts can be executed within a session to perform various actions
such as reading output and issuing commands.

## Types of Scripts
### Native Scripts
Native scripts are written in a language specialized for string
manipulation. These scripts utilize a simple assembly-like language 
where registers are loaded with strings, strings are compared, and output 
is provided accordingly.

### External Scripts
External scripts, also referred to as third-party scripts, are executable
files that can process `stdin` and `stdout`.
These scripts interact with the Provisioner tool by receiving serial input 
and emitting output through stdin and stdout.

## Considerations for Writing Scripts
When writing external scripts for the Provisioner tool, it's essential to
consider the two types of terminal discipline used for sending serial output 
to the script:

* Character-based Scripts: Each character is sent to the script as it arrives 
at the serial port, and the same is done in reverse order.
* Line-based Scripts: The entire line is fetched and sent to the script.

Character-based scripts have the potential to intercept human interactions, 
while line-based scripts may mask these interactions to some extent.

Additionally, external scripts can emit strings on `stderr`. Any strings emitted 
on `stderr` will be logged by the Provisioner tool at the `info` level in the 
application log.
