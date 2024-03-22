MAJOR=$(shell ./maj.sh)
MINOR=$(shell ./min.sh)
CHASH=$(shell git log --pretty=oneline| head -n1 |cut -d" " -f1)

all: upx

provisioner: asymcrypt.go  calendar.go  calendar_utils.go  cmdline.go  commands.go  config.go  escapes.go  http.go  monitor.go  provisioner.go  serial.go  snmp.go  ssh.go  syslog.go  tftp.go
	go build -ldflags "-w -X 'main.Version=$(MAJOR)' -X 'main.Build=$(MINOR)' -X 'main.Hash=$(CHASH)'"
upx: provisioner
	upx provisioner

clean:
	rm provisioner
