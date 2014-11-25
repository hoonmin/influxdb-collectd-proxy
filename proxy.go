package main

import (
	"flag"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"
	"time"

	influxdb "github.com/influxdb/influxdb/client"
	collectd "github.com/paulhammond/gocollectd"
)

const influxWriteInterval = time.Second
const influxWriteLimit = 50

var (
	proxyHost   *string
	proxyPort   *string
	typesdbPath *string
	logPath     *string
	verbose     *bool

	// influxdb options
	host      *string
	username  *string
	password  *string
	database  *string
	normalize *bool
	storeRates *bool

	// Format
	hostnameAsColumn *bool
	
	types       Types
	client      *influxdb.Client
	beforeCache map[string]CacheEntry
)

// point cache to perform data normalization for COUNTER and DERIVE types
type CacheEntry struct {
	Timestamp int64
	Value     float64
	Hostname  string
}

// signal handler
func handleSignals(c chan os.Signal) {
	// block until a signal is received
	sig := <-c

	log.Printf("exit with a signal: %v\n", sig)
	os.Exit(1)
}

func init() {
	// proxy options
	proxyHost = flag.String("proxyhost", "0.0.0.0", "host for proxy")
	proxyPort = flag.String("proxyport", "8096", "port for proxy")
	typesdbPath = flag.String("typesdb", "types.db", "path to Collectd's types.db")
	logPath = flag.String("logfile", "", "path to log file (log to stderr if empty)")
	verbose = flag.Bool("verbose", false, "true if you need to trace the requests")

	// influxdb options
	host = flag.String("influxdb", "localhost:8086", "host:port for influxdb")
	username = flag.String("username", "root", "username for influxdb")
	password = flag.String("password", "root", "password for influxdb")
	database = flag.String("database", "", "database for influxdb")
	normalize = flag.Bool("normalize", true, "true if you need to normalize data for COUNTER types (over time)")
	storeRates = flag.Bool("storerates", true, "true if you need to derive rates from DERIVE types")

	// format options
	hostnameAsColumn = flag.Bool("hostname-as-column", false, "true if you want the hostname as column, not in series name")
	flag.Parse()

	beforeCache = make(map[string]CacheEntry)

	// read types.db
	var err error
	types, err = ParseTypesDB(*typesdbPath)
	if err != nil {
		log.Fatalf("failed to read types.db: %v\n", err)
	}
}

func main() {
	var err error

	if *logPath != "" {
		logFile, err := os.OpenFile(*logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("failed to open file: %v\n", err)
		}
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// make influxdb client
	client, err = influxdb.NewClient(&influxdb.ClientConfig{
		Host:     *host,
		Username: *username,
		Password: *password,
		Database: *database,
	})
	if err != nil {
		log.Fatalf("failed to make a influxdb client: %v\n", err)
	}

	// register a signal handler
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, os.Kill)
	go handleSignals(sc)

	// make channel for collectd
	c := make(chan collectd.Packet)

	// then start to listen
	go collectd.Listen(*proxyHost+":"+*proxyPort, c)
	log.Printf("proxy started on %s:%s\n", *proxyHost, *proxyPort)
	timer := time.Now()
	var seriesGroup []*influxdb.Series
	for {
		packet := <-c
		seriesGroup = append(seriesGroup, processPacket(packet)...)

		if time.Since(timer) < influxWriteInterval && len(seriesGroup) < influxWriteLimit {
			continue
		} else {
			if len(seriesGroup) > 0 {
				if err := client.WriteSeries(seriesGroup); err != nil {
					log.Printf("failed to write series group to influxdb: %s\n", err)
				}
				if *verbose {
					log.Printf("[TRACE] wrote %d series\n", len(seriesGroup))
				}
				seriesGroup = make([]*influxdb.Series, 0)
			}
			timer = time.Now()
		}
	}
}

func processPacket(packet collectd.Packet) []*influxdb.Series {
	if *verbose {
		log.Printf("[TRACE] got a packet: %v\n", packet)
	}

	var seriesGroup []*influxdb.Series
		
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
		nameNoHostname := pluginName + "." + typeName
		// influxdb stuffs
		timestamp := packet.Time().UnixNano() / 1000000
		value := values[i].Float64()
		dataType := packet.DataTypes[i]
		readyToSend := true
		normalizedValue := value

		if *normalize && dataType == collectd.TypeCounter || *storeRates && dataType == collectd.TypeDerive {
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
                        	Hostname:  hostName,
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
			if *hostnameAsColumn {
				series = &influxdb.Series{
                                        Name:    nameNoHostname,
                                        Columns: []string{"time", "value", "hostname"},
                                        Points: [][]interface{}{
                                                []interface{}{timestamp, normalizedValue, hostName},
                                        },
                                }
			}
			if *verbose {
				log.Printf("[TRACE] ready to send series: %v\n", series)
			}
			seriesGroup = append(seriesGroup, series)
		}
	}
	return seriesGroup
}
