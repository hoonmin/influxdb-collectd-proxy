GOPATH:=$(GOPATH):`pwd`
BIN=bin
EXE=proxy

GOCOLLECTD=github.com/paulhammond/gocollectd
INFLUXDBGO=github.com/influxdb/influxdb/client

all: get build

get:
	GOPATH=$(GOPATH) go get $(GOCOLLECTD)
	GOPATH=$(GOPATH) go get $(INFLUXDBGO)

build:
	GOPATH=$(GOPATH) go build -o $(BIN)/$(EXE)

clean: 
	rm -rf src
	rm -rf pkg
	rm -rf bin
