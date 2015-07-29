PWD := $(shell pwd)
BUILDDIR := $(PWD)/build

all: clean install build

build:
	GOPATH=$(BUILDDIR) go get github.com/tools/godep
	GOPATH=$(BUILDDIR) $(BUILDDIR)/bin/godep restore
	GOPATH=$(BUILDDIR) go build -o $(BUILDDIR)/bin/influxdb-collectd-proxy

clean: 
	rm -rf $(BUILDDIR)