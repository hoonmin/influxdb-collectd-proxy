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
  -normalize=true: true if you need to normalize data for COUNTER types (over time)
  -storerates=true: true if you need to derive rates from DERIVE types
  -password="root": password for influxdb
  -proxyhost="0.0.0.0": host for proxy
  -proxyport="8096": port for proxy
  -typesdb="types.db": path to Collectd's types.db
  -username="root": username for influxdb
  -verbose=false: true if you need to trace the requests
```

## Systemd Unit File

Only tested on Arch Linux. You may have to adjust the path of typesdb for your distro.

```
[Unit]
Description=Proxy that forwards collectd data to influxdb

[Service]
Type=simple
ExecStart=/usr/local/bin/influxdb-collectd-proxy --database=collectd --username=root --password=root --typesdb=/usr/share/collectd/types.db
User=collectd-proxy
Group=collectd-proxy

[Install]
RequiredBy=collectd.service
```

## Dependencies

- http://github.com/paulhammond/gocollectd
- http://github.com/influxdb/influxdb/client
  - https://github.com/influxdb/influxdb/tree/master/client

## References

- http://github.com/bpaquet/collectd-influxdb-proxy

## Contributors

This project is maintained with following contributors' supports.

- porjo (http://github.com/porjo)
- feraudet (http://github.com/feraudet)
- falzm (http://github.com/falzm)
- vbatoufflet (http://github.com/vbatoufflet)
- cstorey (http://github.com/cstorey)
- yanfali (http://github.com/yanfali)
- linyanzhong (http://github.com/linyanzhong)
- rplessl (http://github.com/rplessl)
