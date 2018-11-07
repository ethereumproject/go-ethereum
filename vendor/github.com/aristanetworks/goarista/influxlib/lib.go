// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package influxlib

import (
	"errors"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb/client/v2"
)

// Row is defined as a map where the key (string) is the name of the
// column (field name) and the value is left as an interface to
// accept any value.
type Row map[string]interface{}

// InfluxDBConnection is an object that the wrapper uses.
// Holds a client of the type v2.Client and the configuration
type InfluxDBConnection struct {
	Client influxdb.Client
	Config *InfluxConfig
}

// Point represents a datapoint to be written.
// Measurement:
//      The measurement to write to
// Tags:
//      A dictionary of tags in the form string=string
// Fields:
//      A dictionary of fields(keys) with their associated values
type Point struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

// Connect takes an InfluxConfig and establishes a connection
// to InfluxDB. It returns an InfluxDBConnection structure.
// InfluxConfig may be nil for a default connection.
func Connect(config *InfluxConfig) (*InfluxDBConnection, error) {
	var con influxdb.Client
	var err error

	switch config.Protocol {
	case HTTP:
		addr := fmt.Sprintf("http://%s:%v", config.Hostname, config.Port)
		con, err = influxdb.NewHTTPClient(influxdb.HTTPConfig{
			Addr:    addr,
			Timeout: 1 * time.Second,
		})
	case UDP:
		addr := fmt.Sprintf("%s:%v", config.Hostname, config.Port)
		con, err = influxdb.NewUDPClient(influxdb.UDPConfig{
			Addr: addr,
		})
	default:
		return nil, errors.New("Invalid Protocol")
	}

	if err != nil {
		return nil, err
	}

	return &InfluxDBConnection{Client: con, Config: config}, nil
}

// RecordBatchPoints takes in a slice of influxlib.Point and writes them to the
// database.
func (conn *InfluxDBConnection) RecordBatchPoints(points []Point) error {
	var err error
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:        conn.Config.Database,
		Precision:       "ns",
		RetentionPolicy: conn.Config.RetentionPolicy,
	})
	if err != nil {
		return err
	}

	var influxPoints []*influxdb.Point

	for _, p := range points {
		if p.Timestamp.IsZero() {
			p.Timestamp = time.Now()
		}

		point, err := influxdb.NewPoint(p.Measurement, p.Tags, p.Fields,
			p.Timestamp)
		if err != nil {
			return err
		}
		influxPoints = append(influxPoints, point)
	}

	bp.AddPoints(influxPoints)

	if err = conn.Client.Write(bp); err != nil {
		return err
	}

	return nil
}

// WritePoint stores a datapoint to the database.
// Measurement:
//		The measurement to write to
// Tags:
//		A dictionary of tags in the form string=string
// Fields:
//		A dictionary of fields(keys) with their associated values
func (conn *InfluxDBConnection) WritePoint(measurement string,
	tags map[string]string, fields map[string]interface{}) error {
	return conn.RecordPoint(Point{
		Measurement: measurement,
		Tags:        tags,
		Fields:      fields,
		Timestamp:   time.Now(),
	})
}

// RecordPoint implements the same as WritePoint but used a point struct
// as the argument instead.
func (conn *InfluxDBConnection) RecordPoint(p Point) error {
	return conn.RecordBatchPoints([]Point{p})
}

// Query sends a query to the influxCli and returns a slice of
// rows. Rows are of type map[string]interface{}
func (conn *InfluxDBConnection) Query(query string) ([]Row, error) {
	q := influxdb.NewQuery(query, conn.Config.Database, "ns")
	var rows []Row
	var index = 0

	response, err := conn.Client.Query(q)
	if err != nil {
		return nil, err
	}
	if response.Error() != nil {
		return nil, response.Error()
	}

	// The intent here is to combine the separate client v2
	// series into a single array. As a result queries that
	// utilize "group by" will be combined into a single
	// array. And the tag value will be added to the query.
	// Similar to what you would expect from a SQL query
	for _, result := range response.Results {
		for _, series := range result.Series {
			columnNames := series.Columns
			for _, row := range series.Values {
				rows = append(rows, make(Row))
				for columnIdx, value := range row {
					rows[index][columnNames[columnIdx]] = value
				}
				for tagKey, tagValue := range series.Tags {
					rows[index][tagKey] = tagValue
				}
				index++
			}
		}
	}

	return rows, nil
}

// Close closes the connection opened by Connect()
func (conn *InfluxDBConnection) Close() {
	conn.Client.Close()
}
