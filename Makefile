PWD := $(shell pwd)
BUILDDIR := $(PWD)/build

all: clean install build

install:
	GOPATH=$(BUILDDIR) go get github.com/tools/godep
	GOPATH=$(BUILDDIR) $(BUILDDIR)/bin/godep restore

build:
	GOPATH=$(BUILDDIR) go build -o $(BUILDDIR)/bin/influxdb-collectd-proxy

clean: 
	rm -rf $(BUILDDIR)