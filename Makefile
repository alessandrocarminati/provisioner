MAJOR=$(shell ./maj.sh)
MINOR=$(shell ./min.sh)
CHASH=$(shell git log --pretty=oneline| head -n1 |cut -d" " -f1)
DIRTY=$(shell ./dirty.sh)
ifeq ($(shell command -v upx 2> /dev/null),)
	ALL_DEPENDENCIES := provisioner-$(MAJOR).$(MINOR)
else
	ALL_DEPENDENCIES := provisioner.upx-$(MAJOR).$(MINOR)
endif
SOURCES := $(wildcard *.go)

all: $(ALL_DEPENDENCIES)

provisioner-$(MAJOR).$(MINOR): $(SOURCES)
	go build -ldflags "-w -X 'main.Version=$(MAJOR)' -X 'main.Build=$(MINOR)' -X 'main.Hash=$(CHASH)' -X 'main.Dirty=$(DIRTY)'" -o  $(prefix)provisioner-$(MAJOR).$(MINOR)

provisioner.upx-$(MAJOR).$(MINOR): provisioner-$(MAJOR).$(MINOR)
	rm -f $(prefix)provisioner.upx-$(MAJOR).$(MINOR)
	upx  $(prefix)provisioner-$(MAJOR).$(MINOR) -o  $(prefix)provisioner.upx-$(MAJOR).$(MINOR)
	touch $(prefix)provisioner.upx-$(MAJOR).$(MINOR)

clean:
	rm -rf  provisioner-* provisioner.upx-* dist

dist:
	@mkdir -p dist
	@cp config.json dist/config-sample.json
	@echo "put here your google credentials" >dist/cred.json
	@for arch in 386 amd64 arm arm64 mipsle; do \
		$(MAKE) provisioner-$(MAJOR).$(MINOR) GOOS=linux GOARCH=$$arch prefix=dist/$${arch}.; \
		$(MAKE) provisioner.upx-$(MAJOR).$(MINOR) GOOS=linux GOARCH=$$arch prefix=dist/$${arch}.; \
		tar zcf dist/provisioner.$${arch}-$(MAJOR).$(MINOR).tar.gz dist/$${arch}.provisioner.upx-$(MAJOR).$(MINOR) dist/$${arch}.provisioner-$(MAJOR).$(MINOR) dist/config-sample.json dist/cred.json; \
		rm -f dist/$${arch}.provisioner.upx-$(MAJOR).$(MINOR) dist/$${arch}.provisioner-$(MAJOR).$(MINOR); \
	done
	rm dist/config-sample.json dist/cred.json
