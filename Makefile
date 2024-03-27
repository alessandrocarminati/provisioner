
all: bin/rootfs.cpio

bin/init: goinit.go
	GOOS=linux GOARCH=arm64 go build -o bin/init goinit.go
#	aarch64-linux-gnu-gcc -static -o bin/init init.c
#	aarch64-linux-gnu-as init.s  -o init.o
#	aarch64-linux-gnu-ld -o bin/init init.o

bin/rootfs.cpio: rootfs/init
	cd rootfs && find . -print0 | cpio --null --create --verbose --format=newc 2>/dev/null > ../bin/rootfs.cpio && cd ..
#	find rootfs -depth -print0 | cpio --null -ov --format=newc >bin/rootfs.cpio

rootfs/init: bin/init
#	mkdir -p rootfs/sbin
	mkdir rootfs/dev
	sudo mknod rootfs/dev/console c 5 1
	cp bin/init rootfs/init

clean:
	rm -rf bin/*
	rm -rf rootfs/*

