MAJOR=$(shell ./maj.sh)
MINOR=$(shell ./min.sh)
CHASH=$(shell git log --pretty=oneline| head -n1 |cut -d" " -f1)
DIRTY=$(shell ./dirty.sh)
ifeq ($(shell command -v upx 2> /dev/null),)
	ALL_DEPENDENCIES := provisioner
else
	ALL_DEPENDENCIES := provisioner.upx
endif

all: $(ALL_DEPENDENCIES)

provisioner: asymcrypt.go  calendar.go  calendar_utils.go  cmdline.go  commands.go  config.go  escapes.go  http.go  monitor.go  provisioner.go  serial.go  snmp.go  ssh.go  syslog.go  tftp.go
	go build -ldflags "-w -X 'main.Version=$(MAJOR)' -X 'main.Build=$(MINOR)' -X 'main.Hash=$(CHASH)' -X 'main.Dirty=$(DIRTY)'"

provisioner.upx: provisioner
	upx provisioner -o provisioner.upx
	touch provisioner.upx

clean:
	rm -rf  provisioner provisioner.upx
