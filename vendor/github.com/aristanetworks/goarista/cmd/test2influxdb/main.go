// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// test2influxdb writes results from 'go test -json' to an influxdb
// database.
//
// Example usage:
//
//  go test -json | test2influxdb [options...]
//
// Points are written to influxdb with tags:
//
//  package
//  type    "package" for a package result; "test" for a test result
//  Additional tags set by -tags flag
//
// And fields:
//
//  test    string  // "NONE" for whole package results
//  elapsed float64 // in seconds
//  pass    float64 // 1 for PASS, 0 for FAIL
//  Additional fields set by -fields flag
//
// "test" is a field instead of a tag to reduce cardinality of data.
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aristanetworks/glog"
	client "github.com/influxdata/influxdb/client/v2"
)

type tag struct {
	key   string
	value string
}

type tags []tag

func (ts *tags) String() string {
	s := make([]string, len(*ts))
	for i, t := range *ts {
		s[i] = t.key + "=" + t.value
	}
	return strings.Join(s, ",")
}

func (ts *tags) Set(s string) error {
	for _, fieldString := range strings.Split(s, ",") {
		kv := strings.Split(fieldString, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid tag, expecting one '=': %q", fieldString)
		}
		key := strings.TrimSpace(kv[0])
		if key == "" {
			return fmt.Errorf("invalid tag key %q in %q", key, fieldString)
		}
		val := strings.TrimSpace(kv[1])
		if val == "" {
			return fmt.Errorf("invalid tag value %q in %q", val, fieldString)
		}

		*ts = append(*ts, tag{key: key, value: val})
	}
	return nil
}

type field struct {
	key   string
	value interface{}
}

type fields []field

func (fs *fields) String() string {
	s := make([]string, len(*fs))
	for i, f := range *fs {
		var valString string
		switch v := f.value.(type) {
		case bool:
			valString = strconv.FormatBool(v)
		case float64:
			valString = strconv.FormatFloat(v, 'f', -1, 64)
		case int64:
			valString = strconv.FormatInt(v, 10) + "i"
		case string:
			valString = v
		}

		s[i] = f.key + "=" + valString
	}
	return strings.Join(s, ",")
}

func (fs *fields) Set(s string) error {
	for _, fieldString := range strings.Split(s, ",") {
		kv := strings.Split(fieldString, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid field, expecting one '=': %q", fieldString)
		}
		key := strings.TrimSpace(kv[0])
		if key == "" {
			return fmt.Errorf("invalid field key %q in %q", key, fieldString)
		}
		val := strings.TrimSpace(kv[1])
		if val == "" {
			return fmt.Errorf("invalid field value %q in %q", val, fieldString)
		}
		var value interface{}
		var err error
		if value, err = strconv.ParseBool(val); err == nil {
			// It's a bool
		} else if value, err = strconv.ParseFloat(val, 64); err == nil {
			// It's a float64
		} else if value, err = strconv.ParseInt(val[:len(val)-1], 0, 64); err == nil &&
			val[len(val)-1] == 'i' {
			// ints are suffixed with an "i"
		} else {
			value = val
		}

		*fs = append(*fs, field{key: key, value: value})
	}
	return nil
}

var (
	flagAddr        = flag.String("addr", "http://localhost:8086", "adddress of influxdb database")
	flagDB          = flag.String("db", "gotest", "use `database` in influxdb")
	flagMeasurement = flag.String("m", "result", "`measurement` used in influxdb database")

	flagTags   tags
	flagFields fields
)

func init() {
	flag.Var(&flagTags, "tags", "set additional `tags`. Ex: name=alice,food=pasta")
	flag.Var(&flagFields, "fields", "set additional `fields`. Ex: id=1234i,long=34.123,lat=72.234")
}

func main() {
	flag.Parse()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: *flagAddr,
	})
	if err != nil {
		glog.Fatal(err)
	}

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{Database: *flagDB})
	if err != nil {
		glog.Fatal(err)
	}

	if err := parseTestOutput(os.Stdin, batch); err != nil {
		glog.Fatal(err)
	}

	if err := c.Write(batch); err != nil {
		glog.Fatal(err)
	}
}

// See https://golang.org/cmd/test2json/ for a description of 'go test
// -json' output
type testEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

func createTags(e *testEvent) map[string]string {
	tags := make(map[string]string, len(flagTags)+2)
	for _, t := range flagTags {
		tags[t.key] = t.value
	}
	resultType := "test"
	if e.Test == "" {
		resultType = "package"
	}
	tags["package"] = e.Package
	tags["type"] = resultType
	return tags
}

func createFields(e *testEvent) map[string]interface{} {
	fields := make(map[string]interface{}, len(flagFields)+3)
	for _, f := range flagFields {
		fields[f.key] = f.value
	}
	// Use a float64 instead of a bool to be able to SUM test
	// successes in influxdb.
	var pass float64
	if e.Action == "pass" {
		pass = 1
	}
	fields["pass"] = pass
	fields["elapsed"] = e.Elapsed
	if e.Test != "" {
		fields["test"] = e.Test
	}
	return fields
}

func parseTestOutput(r io.Reader, batch client.BatchPoints) error {
	// pkgs holds packages seen in r. Unfortunately, if a test panics,
	// then there is no "fail" result from a package. To detect these
	// kind of failures, keep track of all the packages that never had
	// a "pass" or "fail".
	//
	// The last seen timestamp is stored with the package, so that
	// package result measurement written to influxdb can be later
	// than any test result for that package.
	pkgs := make(map[string]time.Time)
	d := json.NewDecoder(r)
	for {
		e := &testEvent{}
		if err := d.Decode(e); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		switch e.Action {
		case "pass", "fail":
		default:
			continue
		}

		if e.Test == "" {
			// A package has completed.
			delete(pkgs, e.Package)
		} else {
			pkgs[e.Package] = e.Time
		}

		point, err := client.NewPoint(
			*flagMeasurement,
			createTags(e),
			createFields(e),
			e.Time,
		)
		if err != nil {
			return err
		}

		batch.AddPoint(point)
	}

	for pkg, t := range pkgs {
		pkgFail := &testEvent{
			Action:  "fail",
			Package: pkg,
		}
		point, err := client.NewPoint(
			*flagMeasurement,
			createTags(pkgFail),
			createFields(pkgFail),
			// Fake a timestamp that is later than anything that
			// occurred for this package
			t.Add(time.Millisecond),
		)
		if err != nil {
			return err
		}

		batch.AddPoint(point)
	}

	return nil
}
