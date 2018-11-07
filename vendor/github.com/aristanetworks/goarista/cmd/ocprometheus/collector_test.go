// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aristanetworks/goarista/test"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/prometheus/client_golang/prometheus"
)

func makeMetrics(cfg *Config, expValues map[source]float64, notification *pb.Notification,
	prevMetrics map[source]*labelledMetric) map[source]*labelledMetric {

	expMetrics := map[source]*labelledMetric{}
	if prevMetrics != nil {
		expMetrics = prevMetrics
	}
	for src, v := range expValues {
		metric := cfg.getMetricValues(src)
		if metric == nil || metric.desc == nil || metric.labels == nil {
			panic("cfg.getMetricValues returned nil")
		}
		// Preserve current value of labels
		labels := metric.labels
		if _, ok := expMetrics[src]; ok && expMetrics[src] != nil {
			labels = expMetrics[src].labels
		}

		// Handle string updates
		if notification.Update != nil {
			if update, err := findUpdate(notification, src.path); err == nil {
				val, _, ok := parseValue(update)
				if !ok {
					continue
				}
				if strVal, ok := val.(string); ok {
					if !metric.stringMetric {
						continue
					}
					v = metric.defaultValue
					labels[len(labels)-1] = strVal
				}
			}
		}
		expMetrics[src] = &labelledMetric{
			metric: prometheus.MustNewConstMetric(metric.desc, prometheus.GaugeValue, v,
				labels...),
			labels:       labels,
			defaultValue: metric.defaultValue,
			stringMetric: metric.stringMetric,
		}
	}
	// Handle deletion
	for key := range expMetrics {
		if _, ok := expValues[key]; !ok {
			delete(expMetrics, key)
		}
	}
	return expMetrics
}

func findUpdate(notif *pb.Notification, path string) (*pb.Update, error) {
	prefix := notif.Prefix.Element
	for _, v := range notif.Update {
		fullPath := "/" + strings.Join(append(prefix, v.Path.Element...), "/")
		if strings.Contains(path, fullPath) || path == fullPath {
			return v, nil
		}
	}
	return nil, fmt.Errorf("Failed to find matching update for path %v", path)
}

func makeResponse(notif *pb.Notification) *pb.SubscribeResponse {
	return &pb.SubscribeResponse{
		Response: &pb.SubscribeResponse_Update{Update: notif},
	}
}

func TestUpdate(t *testing.T) {
	config := []byte(`
devicelabels:
        10.1.1.1:
                lab1: val1
                lab2: val2
        '*':
                lab1: val3
                lab2: val4
subscriptions:
        - /Sysdb/environment/cooling/status
        - /Sysdb/environment/power/status
        - /Sysdb/bridging/igmpsnooping/forwarding/forwarding/status
metrics:
        - name: fanName
          path: /Sysdb/environment/cooling/status/fan/name
          help: Fan Name
          valuelabel: name
          defaultvalue: 2.5
        - name: intfCounter
          path: /Sysdb/(lag|slice/phy/.+)/intfCounterDir/(?P<intf>.+)/intfCounter
          help: Per-Interface Bytes/Errors/Discards Counters
        - name: fanSpeed
          path: /Sysdb/environment/cooling/status/fan/speed/value
          help: Fan Speed
        - name: igmpSnoopingInf
          path: /Sysdb/igmpsnooping/vlanStatus/(?P<vlan>.+)/ethGroup/(?P<mac>.+)/intf/(?P<intf>.+)
          help: IGMP snooping status`)
	cfg, err := parseConfig(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	coll := newCollector(cfg)

	notif := &pb.Notification{
		Prefix: &pb.Path{Element: []string{"Sysdb"}},
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Element: []string{"lag", "intfCounterDir", "Ethernet1", "intfCounter"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("42"),
				},
			},
			{
				Path: &pb.Path{
					Element: []string{"environment", "cooling", "status", "fan", "speed"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("{\"value\": 45}"),
				},
			},
			{
				Path: &pb.Path{
					Element: []string{"igmpsnooping", "vlanStatus", "2050", "ethGroup",
						"01:00:5e:01:01:01", "intf", "Cpu"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("true"),
				},
			},
			{
				Path: &pb.Path{
					Element: []string{"environment", "cooling", "status", "fan", "name"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("\"Fan1.1\""),
				},
			},
		},
	}
	expValues := map[source]float64{
		{
			addr: "10.1.1.1",
			path: "/Sysdb/lag/intfCounterDir/Ethernet1/intfCounter",
		}: 42,
		{
			addr: "10.1.1.1",
			path: "/Sysdb/environment/cooling/status/fan/speed/value",
		}: 45,
		{
			addr: "10.1.1.1",
			path: "/Sysdb/igmpsnooping/vlanStatus/2050/ethGroup/01:00:5e:01:01:01/intf/Cpu",
		}: 1,
		{
			addr: "10.1.1.1",
			path: "/Sysdb/environment/cooling/status/fan/name",
		}: 2.5,
	}

	coll.update("10.1.1.1:6042", makeResponse(notif))
	expMetrics := makeMetrics(cfg, expValues, notif, nil)
	if !test.DeepEqual(expMetrics, coll.metrics) {
		t.Errorf("Mismatched metrics: %v", test.Diff(expMetrics, coll.metrics))
	}

	// Update two values, and one path which is not a metric
	notif = &pb.Notification{
		Prefix: &pb.Path{Element: []string{"Sysdb"}},
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Element: []string{"lag", "intfCounterDir", "Ethernet1", "intfCounter"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("52"),
				},
			},
			{
				Path: &pb.Path{
					Element: []string{"environment", "cooling", "status", "fan", "name"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("\"Fan2.1\""),
				},
			},
			{
				Path: &pb.Path{
					Element: []string{"environment", "doesntexist", "status"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("{\"value\": 45}"),
				},
			},
		},
	}
	src := source{
		addr: "10.1.1.1",
		path: "/Sysdb/lag/intfCounterDir/Ethernet1/intfCounter",
	}
	expValues[src] = 52

	coll.update("10.1.1.1:6042", makeResponse(notif))
	expMetrics = makeMetrics(cfg, expValues, notif, expMetrics)
	if !test.DeepEqual(expMetrics, coll.metrics) {
		t.Errorf("Mismatched metrics: %v", test.Diff(expMetrics, coll.metrics))
	}

	// Same path, different device
	notif = &pb.Notification{
		Prefix: &pb.Path{Element: []string{"Sysdb"}},
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Element: []string{"lag", "intfCounterDir", "Ethernet1", "intfCounter"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("42"),
				},
			},
		},
	}
	src.addr = "10.1.1.2"
	expValues[src] = 42

	coll.update("10.1.1.2:6042", makeResponse(notif))
	expMetrics = makeMetrics(cfg, expValues, notif, expMetrics)
	if !test.DeepEqual(expMetrics, coll.metrics) {
		t.Errorf("Mismatched metrics: %v", test.Diff(expMetrics, coll.metrics))
	}

	// Delete a path
	notif = &pb.Notification{
		Prefix: &pb.Path{Element: []string{"Sysdb"}},
		Delete: []*pb.Path{
			{
				Element: []string{"lag", "intfCounterDir", "Ethernet1", "intfCounter"},
			},
		},
	}
	src.addr = "10.1.1.1"
	delete(expValues, src)

	coll.update("10.1.1.1:6042", makeResponse(notif))
	expMetrics = makeMetrics(cfg, expValues, notif, expMetrics)
	if !test.DeepEqual(expMetrics, coll.metrics) {
		t.Errorf("Mismatched metrics: %v", test.Diff(expMetrics, coll.metrics))
	}

	// Non-numeric update to path without value label
	notif = &pb.Notification{
		Prefix: &pb.Path{Element: []string{"Sysdb"}},
		Update: []*pb.Update{
			{
				Path: &pb.Path{
					Element: []string{"lag", "intfCounterDir", "Ethernet1", "intfCounter"},
				},
				Value: &pb.Value{
					Type:  pb.Encoding_JSON,
					Value: []byte("\"test\""),
				},
			},
		},
	}

	coll.update("10.1.1.1:6042", makeResponse(notif))
	// Don't make new metrics as it should have no effect
	if !test.DeepEqual(expMetrics, coll.metrics) {
		t.Errorf("Mismatched metrics: %v", test.Diff(expMetrics, coll.metrics))
	}
}
