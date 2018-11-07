// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package influxlib

//Connection type.
const (
	HTTP = "HTTP"
	UDP  = "UDP"
)

//InfluxConfig is a configuration struct for influxlib.
type InfluxConfig struct {
	Hostname        string
	Port            uint16
	Protocol        string
	Database        string
	RetentionPolicy string
}
