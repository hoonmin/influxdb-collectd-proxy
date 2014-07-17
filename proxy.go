package main

import (
	"flag"
	influxdb "github.com/influxdb/influxdb/client"
	collectd "github.com/paulhammond/gocollectd"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
)

// proxy options
var proxyPort = flag.String("proxyport", "8096", "port for proxy")
var typesdbPath = flag.String("typesdb", "types.db", "path to types.db")
var logPath = flag.String("logfile", "proxy.log", "path to log file")
var verbose = flag.Bool("verbose", false, "true if you need to trace the requests")

// influxdb options
var host = flag.String("influxdb", "localhost:8086", "host:port for influxdb")
var username = flag.String("username", "root", "username for influxdb")
var password = flag.String("password", "root", "password for influxdb")
var database = flag.String("database", "", "database for influxdb")
var normalize = flag.Bool("normalize", true, "true if you need to normalize data for COUNTER and DERIVE types (over time)")

// point cache to perform data normalization for COUNTER and DERIVE types
type CacheEntry struct {
	Timestamp int64
	Value     float64
}

var beforeCache = make(map[string]CacheEntry)

// signal handler
func handleSignals(c chan os.Signal) {
	// block until a signal is received
	sig := <-c

	log.Printf("exit with a signal: %v\n", sig)
	os.Exit(1)
}

func main() {
	flag.Parse()

	// log file
	logFile, err := os.OpenFile(*logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open file: %v\n", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// read types.db
	types, err := ParseTypesDB(*typesdbPath)
	if err != nil {
		log.Fatalf("failed to read types.db: %v\n", err)
		return
	}

	// make influxdb client
	client, err := influxdb.NewClient(&influxdb.ClientConfig{
		Host:     *host,
		Username: *username,
		Password: *password,
		Database: *database,
	})
	if err != nil {
		log.Fatalf("failed to make a influxdb client: %v\n", err)
		return
	}

	// register a signal handler
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)
	go handleSignals(sc)

	// make channel for collectd
	c := make(chan collectd.Packet)

	// then start to listen
	go collectd.Listen("0.0.0.0:"+*proxyPort, c)
	log.Printf("proxy started on %s\n", *proxyPort)
	for {
		packet := <-c
		if *verbose {
			log.Printf("[TRACE] got a packet: %v\n", packet.Hostname, packet)
		}

		// for all metrics in the packet
		for i, _ := range packet.ValueNames() {
			values, _ := packet.ValueNumbers()

			// get a type for this packet
			t := types[packet.Type]

			// pass the unknowns
			if t == nil && packet.TypeInstance == "" {
				log.Printf("unknown type instance on %s\n", packet.Plugin)
				continue
			}

			// as hostname contains commas, let's replace them
			hostName := strings.Replace(packet.Hostname, ".", "_", -1)

			// if there's a PluginInstance, use it
			pluginName := packet.Plugin
			if packet.PluginInstance != "" {
				pluginName += "-" + packet.PluginInstance
			}

			// if there's a TypeInstance, use it
			typeName := packet.Type
			if packet.TypeInstance != "" {
				typeName += "-" + packet.TypeInstance
			} else if t != nil {
				typeName += "-" + t[i][0]
			}

			name := hostName + "." + pluginName + "." + typeName

			// influxdb stuffs
			timestamp := packet.Time().UnixNano() / 1000000
			value := values[i].Float64()
			dataType := packet.DataTypes[i]
			readyToSend := true
			normalizedValue := value

			if *normalize && dataType == collectd.TypeCounter || dataType == collectd.TypeDerive {
				if before, ok := beforeCache[name]; ok && before.Value != math.NaN() {
					// normalize over time
					if timestamp-before.Timestamp > 0 {
						normalizedValue = (value - before.Value) / float64((timestamp-before.Timestamp)/1000)
					} else {
						normalizedValue = value - before.Value
					}
				} else {
					// skip current data if there's no initial entry
					readyToSend = false
				}
				entry := CacheEntry{
					Timestamp: timestamp,
					Value:     value,
				}
				beforeCache[name] = entry
			}

			if readyToSend {
				series := &influxdb.Series{
					Name:    name,
					Columns: []string{"time", "value"},
					Points: [][]interface{}{
						[]interface{}{timestamp, normalizedValue},
					},
				}

				if *verbose {
					log.Printf("[TRACE] ready to send series: %v\n", series)
				}

				if err := client.WriteSeries([]*influxdb.Series{series}); err != nil {
					log.Printf("failed to write in influxdb: %s -> %v, reason=%s\n", name, values[i], err)
					continue
				}
			}
		}
	}
}
