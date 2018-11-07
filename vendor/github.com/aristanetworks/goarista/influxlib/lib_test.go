// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package influxlib

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testFields(line string, fields map[string]interface{},
	t *testing.T) {
	for k, v := range fields {
		formatString := "%s=%v"

		if _, ok := v.(string); ok {
			formatString = "%s=%q"
		}
		assert.Contains(t, line, fmt.Sprintf(formatString, k, v),
			fmt.Sprintf(formatString+" expected in %s", k, v, line))
	}
}

func testTags(line string, tags map[string]string,
	t *testing.T) {
	for k, v := range tags {
		assert.Contains(t, line, fmt.Sprintf("%s=%s", k, v),
			fmt.Sprintf("%s=%s expected in %s", k, v, line))
	}
}

func TestBasicWrite(t *testing.T) {
	testConn, _ := NewMockConnection()

	measurement := "TestData"
	tags := map[string]string{
		"tag1": "Happy",
		"tag2": "Valentines",
		"tag3": "Day",
	}
	fields := map[string]interface{}{
		"Data1": 1234,
		"Data2": "apples",
		"Data3": 5.34,
	}

	err := testConn.WritePoint(measurement, tags, fields)
	assert.NoError(t, err)

	line, err := GetTestBuffer(testConn)
	assert.NoError(t, err)

	assert.Contains(t, line, measurement,
		fmt.Sprintf("%s does not appear in %s", measurement, line))
	testTags(line, tags, t)
	testFields(line, fields, t)
}

func TestConnectionToHostFailure(t *testing.T) {
	assert := assert.New(t)
	var err error

	config := &InfluxConfig{
		Port:     8086,
		Protocol: HTTP,
		Database: "test",
	}
	config.Hostname = "this is fake.com"
	_, err = Connect(config)
	assert.Error(err)
	config.Hostname = "\\-Fake.Url.Com"
	_, err = Connect(config)
	assert.Error(err)
}

func TestWriteFailure(t *testing.T) {
	con, _ := NewMockConnection()

	measurement := "TestData"
	tags := map[string]string{
		"tag1": "hi",
	}
	data := map[string]interface{}{
		"Data1": "cats",
	}
	err := con.WritePoint(measurement, tags, data)
	assert.NoError(t, err)

	fc, _ := con.Client.(*fakeClient)
	fc.failAll = true

	err = con.WritePoint(measurement, tags, data)
	assert.Error(t, err)
}

func TestQuery(t *testing.T) {
	query := "SELECT * FROM 'system' LIMIT 50;"

	con, _ := NewMockConnection()

	_, err := con.Query(query)

	assert.NoError(t, err)
}

func TestAddAndWriteBatchPoints(t *testing.T) {
	testConn, _ := NewMockConnection()

	measurement := "TestData"
	points := []Point{
		Point{
			Measurement: measurement,
			Tags: map[string]string{
				"tag1": "Happy",
				"tag2": "Valentines",
				"tag3": "Day",
			},
			Fields: map[string]interface{}{
				"Data1": 1234,
				"Data2": "apples",
				"Data3": 5.34,
			},
			Timestamp: time.Now(),
		},
		Point{
			Measurement: measurement,
			Tags: map[string]string{
				"tag1": "Happy",
				"tag2": "New",
				"tag3": "Year",
			},
			Fields: map[string]interface{}{
				"Data1": 5678,
				"Data2": "bananas",
				"Data3": 3.14,
			},
			Timestamp: time.Now(),
		},
	}

	err := testConn.RecordBatchPoints(points)
	assert.NoError(t, err)

	line, err := GetTestBuffer(testConn)
	assert.NoError(t, err)

	assert.Contains(t, line, measurement,
		fmt.Sprintf("%s does not appear in %s", measurement, line))
	for _, p := range points {
		testTags(line, p.Tags, t)
		testFields(line, p.Fields, t)
	}
}
