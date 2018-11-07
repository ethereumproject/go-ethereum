// Copyright (c) 2016 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// The octsdb tool pushes OpenConfig telemetry to OpenTSDB.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aristanetworks/goarista/gnmi"

	"github.com/aristanetworks/glog"
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

func main() {

	// gNMI options
	cfg := &gnmi.Config{}
	flag.StringVar(&cfg.Addr, "addr", "localhost", "gNMI gRPC server `address`")
	flag.StringVar(&cfg.CAFile, "cafile", "", "Path to server TLS certificate file")
	flag.StringVar(&cfg.CertFile, "certfile", "", "Path to client TLS certificate file")
	flag.StringVar(&cfg.KeyFile, "keyfile", "", "Path to client TLS private key file")
	flag.StringVar(&cfg.Username, "username", "", "Username to authenticate with")
	flag.StringVar(&cfg.Password, "password", "", "Password to authenticate with")
	flag.BoolVar(&cfg.TLS, "tls", false, "Enable TLS")

	// Program options
	subscribePaths := flag.String("paths", "/", "Comma-separated list of paths to subscribe to")

	tsdbFlag := flag.String("tsdb", "",
		"Address of the OpenTSDB server where to push telemetry to")
	textFlag := flag.Bool("text", false,
		"Print the output as simple text")
	configFlag := flag.String("config", "",
		"Config to turn OpenConfig telemetry into OpenTSDB put requests")
	isUDPServerFlag := flag.Bool("isudpserver", false,
		"Set to true to run as a UDP to TCP to OpenTSDB server.")
	udpAddrFlag := flag.String("udpaddr", "",
		"Address of the UDP server to connect to/serve on.")
	parityFlag := flag.Int("parityshards", 0,
		"Number of parity shards for the Reed Solomon Erasure Coding used for UDP."+
			" Clients and servers should have the same number.")
	udpTimeoutFlag := flag.Duration("udptimeout", 2*time.Second,
		"Timeout for each")

	flag.Parse()
	if !(*tsdbFlag != "" || *textFlag || *udpAddrFlag != "") {
		glog.Fatal("Specify the address of the OpenTSDB server to write to with -tsdb")
	} else if *configFlag == "" {
		glog.Fatal("Specify a JSON configuration file with -config")
	}

	config, err := loadConfig(*configFlag)
	if err != nil {
		glog.Fatal(err)
	}
	// Ignore the default "subscribe-to-everything" subscription of the
	// -subscribe flag.
	subscriptions := strings.Split(*subscribePaths, ",")
	if subscriptions[0] == "" {
		subscriptions = subscriptions[1:]
	}
	// Add the subscriptions from the config file.
	subscriptions = append(subscriptions, config.Subscriptions...)

	// Run a UDP server that forwards messages to OpenTSDB via Telnet (TCP)
	if *isUDPServerFlag {
		if *udpAddrFlag == "" {
			glog.Fatal("Specify the address for the UDP server to listen on with -udpaddr")
		}
		server, err := newUDPServer(*udpAddrFlag, *tsdbFlag, *parityFlag)
		if err != nil {
			glog.Fatal("Failed to create UDP server: ", err)
		}
		glog.Fatal(server.Run())
	}

	var c OpenTSDBConn
	if *textFlag {
		c = newTextDumper()
	} else if *udpAddrFlag != "" {
		c = newUDPClient(*udpAddrFlag, *parityFlag, *udpTimeoutFlag)
	} else {
		// TODO: support HTTP(S).
		c = newTelnetClient(*tsdbFlag)
	}
	ctx := gnmi.NewContext(context.Background(), cfg)
	client, err := gnmi.Dial(cfg)
	if err != nil {
		glog.Fatal(err)
	}
	respChan := make(chan *pb.SubscribeResponse)
	errChan := make(chan error)
	subscribeOptions := &gnmi.SubscribeOptions{
		Mode:       "stream",
		StreamMode: "target_defined",
		Paths:      gnmi.SplitPaths(subscriptions),
	}
	go gnmi.Subscribe(ctx, client, subscribeOptions, respChan, errChan)
	for {
		select {
		case resp := <-respChan:
			pushToOpenTSDB(cfg.Addr, c, config, resp.GetUpdate())
		case err := <-errChan:
			glog.Fatal(err)
		}
	}
}

func pushToOpenTSDB(addr string, conn OpenTSDBConn, config *Config, notif *pb.Notification) {
	if notif == nil {
		glog.Error("Nil notification ignored")
		return
	}
	if notif.Timestamp <= 0 {
		glog.Fatalf("Invalid timestamp %d in %s", notif.Timestamp, notif)
	}
	host := addr[:strings.IndexRune(addr, ':')]
	if host == "localhost" {
		// TODO: On Linux this reads /proc/sys/kernel/hostname each time,
		// which isn't the most efficient, but at least we don't have to
		// deal with detecting hostname changes.
		host, _ = os.Hostname()
		if host == "" {
			glog.Info("could not figure out localhost's hostname")
			return
		}
	}
	prefix := "/" + strings.Join(notif.Prefix.Element, "/")
	for _, update := range notif.Update {
		if update.Value == nil || update.Value.Type != pb.Encoding_JSON {
			glog.V(9).Infof("Ignoring incompatible update value in %s", update)
			continue
		}
		value := parseValue(update)
		if value == nil {
			continue
		}
		path := prefix + "/" + strings.Join(update.Path.Element, "/")
		metricName, tags := config.Match(path)
		if metricName == "" {
			glog.V(8).Infof("Ignoring unmatched update at %s: %+v", path, update.Value)
			continue
		}
		tags["host"] = host
		for i, v := range value {
			if len(value) > 1 {
				tags["index"] = strconv.Itoa(i)
			}
			err := conn.Put(&DataPoint{
				Metric:    metricName,
				Timestamp: uint64(notif.Timestamp),
				Value:     v,
				Tags:      tags,
			})
			if err != nil {
				glog.Info("Failed to put datapoint: ", err)
			}
		}
	}
}

// parseValue returns either an integer/floating point value of the given update, or if
// the value is a slice of integers/floating point values. If the value is neither of these
// or if any element in the slice is non numerical, parseValue returns nil.
func parseValue(update *pb.Update) []interface{} {
	var value interface{}

	decoder := json.NewDecoder(bytes.NewReader(update.Value.Value))
	decoder.UseNumber()
	err := decoder.Decode(&value)
	if err != nil {
		glog.Fatalf("Malformed JSON update %q in %s", update.Value.Value, update)
	}

	switch value := value.(type) {
	case json.Number:
		return []interface{}{parseNumber(value, update)}
	case []interface{}:
		for i, val := range value {
			jsonNum, ok := val.(json.Number)
			if !ok {
				// If any value is not a number, skip it.
				glog.Infof("Element %d: %v is %T, not json.Number", i, val, val)
				continue
			}
			num := parseNumber(jsonNum, update)
			value[i] = num
		}
		return value
	case map[string]interface{}:
		// Special case for simple value types that just have a "value"
		// attribute (common case).
		if val, ok := value["value"].(json.Number); ok && len(value) == 1 {
			return []interface{}{parseNumber(val, update)}
		}
	default:
		glog.V(9).Infof("Ignoring non-numeric or non-numeric slice value in %s", update)
	}
	return nil
}

// Convert our json.Number to either an int64, uint64, or float64.
func parseNumber(num json.Number, update *pb.Update) interface{} {
	var value interface{}
	var err error
	if value, err = num.Int64(); err != nil {
		// num is either a large unsigned integer or a floating point.
		if strings.Contains(err.Error(), "value out of range") { // Sigh.
			value, err = strconv.ParseUint(num.String(), 10, 64)
		} else {
			value, err = num.Float64()
			if err != nil {
				glog.Fatalf("Malformed JSON number %q in %s", num, update)
			}
		}
	}
	return value
}
