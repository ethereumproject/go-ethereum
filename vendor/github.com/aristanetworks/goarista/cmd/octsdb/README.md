# octsdb

This is a client for the OpenConfig gRPC interface that pushes telemetry to
OpenTSDB.  Non-numerical data isn't supported by OpenTSDB and is silently
dropped.

This tool requires a config file to specify how to map the path of the
notificatons coming out of the OpenConfig gRPC interface onto OpenTSDB
metric names, and how to extract tags from the path.

## Getting Started
To begin, a list of subscriptions is required (excerpt from `sampleconfig.json`):

```json
	"subscriptions": [
		"/Sysdb/interface/counter/eth/lag",
		"/Sysdb/interface/counter/eth/slice/phy",

		"/Sysdb/environment/temperature/status",
		"/Sysdb/environment/cooling/status",
		"/Sysdb/environment/power/status",

		"/Sysdb/hardware/xcvr/status/all/xcvrStatus"
	],
	...
```

Note that subscriptions should not end with a trailing `/` as that will cause
the subscription to fail.

Afterwards, the metrics are defined (excerpt from `sampleconfig.json`):

```json
	"metrics": {
		"tempSensor": {
			"path": "/Sysdb/(environment)/temperature/status/tempSensor/(?P<sensor>.+)/((?:maxT|t)emperature)"
		},
		...
	}
```

In the metrics path, unnamed matched groups are used to make up the metric name, and named matched groups
are used to extract optional tags. Note that unnamed groups are required, otherwise the metric
name will be empty and the update will be silently dropped.

For example, using the above metrics path applied to an update for the path
`/Sysdb/environment/temperature/status/tempSensor/TempSensor1/temperature`
will lead to the metric name `environment.temperature` and tags `sensor=TempSensor1`.

## Usage

See the `-help` output, but here's an example to push all the metrics defined
in the sample config file:
```
octsdb -addr <switch-hostname>:6042 -config sampleconfig.json -text | nc <tsd-hostname> 4242
```
