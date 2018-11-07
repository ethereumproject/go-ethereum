// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package influxlib

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	influx "github.com/influxdata/influxdb/client/v2"
)

// This will serve as a fake client object to test off of.
// The idea is to provide a way to test the Influx Wrapper
// without having it connected to the database.
type fakeClient struct {
	writer  bytes.Buffer
	failAll bool
}

func (w *fakeClient) Ping(timeout time.Duration) (time.Duration,
	string, error) {
	return 0, "", nil
}

func (w *fakeClient) Query(q influx.Query) (*influx.Response, error) {
	if w.failAll {
		return nil, errors.New("quering points failed")
	}

	return &influx.Response{Results: nil, Err: ""}, nil
}

func (w *fakeClient) Close() error {
	return nil
}

func (w *fakeClient) Write(bp influx.BatchPoints) error {
	if w.failAll {
		return errors.New("writing point failed")
	}
	w.writer.Reset()
	for _, p := range bp.Points() {
		fmt.Fprintf(&w.writer, p.String()+"\n")
	}
	return nil
}

func (w *fakeClient) qString() string {
	return w.writer.String()
}

/***************************************************/

// NewMockConnection returns an influxDBConnection with
// a "fake" client for offline testing.
func NewMockConnection() (*InfluxDBConnection, error) {
	client := new(fakeClient)
	config := &InfluxConfig{
		Hostname: "localhost",
		Port:     8086,
		Protocol: HTTP,
		Database: "Test",
	}
	return &InfluxDBConnection{client, config}, nil
}

// GetTestBuffer returns the string that would normally
// be written to influx DB
func GetTestBuffer(con *InfluxDBConnection) (string, error) {
	fc, ok := con.Client.(*fakeClient)
	if !ok {
		return "", errors.New("Expected a fake client but recieved a real one")
	}
	return fc.qString(), nil
}
