
all: upx

provisioner: asymcrypt.go  calendar.go  calendar_utils.go  cmdline.go  commands.go  config.go  escapes.go  http.go  monitor.go  provisioner.go  serial.go  snmp.go  ssh.go  syslog.go  tftp.go
	go build -ldflags "-w"
upx: provisioner
	upx provisioner

clean:
	rm provisioner
