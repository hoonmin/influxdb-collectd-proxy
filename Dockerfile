FROM golang:1.4.1
RUN go get github.com/tools/godep
ADD . /go/src/app
WORKDIR /go/src/app
RUN godep restore
RUN make
RUN wget -O types.db https://raw.githubusercontent.com/astro/collectd/master/src/types.db
EXPOSE 8096
ENTRYPOINT ["/go/src/app/bin/influxdb-collectd-proxy"]
