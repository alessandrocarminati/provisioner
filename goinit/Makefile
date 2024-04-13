MAJOR ?= $(shell ../verscripts/maj.sh)
MINOR ?= $(shell ../verscripts/min.sh)
CHASH ?= $(shell git log --pretty=oneline| head -n1 |cut -d" " -f1)
DIRTY ?= $(shell ../verscripts/dirty.sh)
GOINIT_ARCH ?= arm64
GOOS ?= linux
GOARCH ?= $(GOINIT_ARCH)

all: bin/$(GOARCH).rootfs-$(MAJOR).$(MINOR).cpio

bin/init: goinit.go
	@echo $(GOOS)
	@echo $(GOARCH)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-w -X 'main.Version=$(MAJOR)' -X 'main.Build=$(MINOR)' -X 'main.Hash=$(CHASH)' -X 'main.Dirty=$(DIRTY)'"  -o bin/init

bin/$(GOARCH).rootfs-$(MAJOR).$(MINOR).cpio: rootfs/init
	cd rootfs && find . -print0 | cpio --null --create --verbose --format=newc 2>/dev/null > ../bin/$(GOARCH).rootfs-$(MAJOR).$(MINOR).cpio && cd ..

rootfs/init: bin/init
	mkdir rootfs/dev
	sudo mknod rootfs/dev/console c 5 1
	cp bin/init rootfs/init

clean:
	rm -rf bin/init  bin/rootfs*.cpio
	rm -rf rootfs/dev  rootfs/init

