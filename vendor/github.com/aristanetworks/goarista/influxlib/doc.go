// Copyright (c) 2018 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

/*
   Package: influxlib
   Title: Influx DB Library
   Authors: ssdaily, manojm321, senkrish, kthommandra
   Email: influxdb-dev@arista.com

   Description: The main purpose of influxlib is to provide users with a simple
   and easy interface through which to connect to influxdb. It removed a lot of
   the need to run the same setup and tear down code to connect the the service.

   Example Code:

   connection, err := influxlib.Connect(&influxlib.InfluxConfig {
      Hostname: conf.Host,
      Port: conf.Port,
      Protocol: influxlib.UDP,
      Database, conf.AlertDB,
   })

   tags := map[string]string {
      "tag1": someStruct.Tag["host"],
      "tag2": someStruct.Tag["tag2"],
   }

   fields := map[string]interface{} {
      "field1": someStruct.Somefield,
      "field2": someStruct.Somefield2,
   }

   connection.WritePoint("measurement", tags, fields)
*/

package influxlib
