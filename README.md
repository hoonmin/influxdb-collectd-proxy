influxdb-collectd-proxy
=======================

A very simple proxy between collectd and influxdb.

## Build

Clone this project and just make it.

```
$ make
```

## Usage

First, add following lines to collectd.conf then restart the collectd daemon.

```
LoadPlugin network

<Plugin network>
  # proxy address
  Server "127.0.0.1" "8096"
</Plugin>
```

And start the proxy.

```
$ bin/proxy --typesdb="types.db" --database="collectd" --username="collectd" --password="collectd"
```

## Options

```
$ bin/proxy --help
Usage of bin/proxy:
  -database="": database for influxdb
  -influxdb="localhost:8086": host:port for influxdb
  -logfile="proxy.log": path to log file
  -normalize=true: true if you need to normalize data for COUNTER and DERIVE types (over time)
  -password="root": password for influxdb
  -proxyport="8096": port for proxy
  -typesdb="types.db": path to Collectd's types.db
  -username="root": username for influxdb
  -verbose=false: true if you need to trace the requests
```

## Dependencies

- http://github.com/paulhammond/gocollectd/
- http://github.com/influxdb/influxdb-go/

## References

- http://github.com/bpaquet/collectd-influxdb-proxy
