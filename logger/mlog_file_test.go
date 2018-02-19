package logger

import (
	"testing"
)

// mlogNeighborsHandleFrom is called once for each neighbors request from a node FROM
var mlogExampleT = &MLogT{
	Description: `Struct for testing mlog structs.`,
	Receiver:    "TESTER",
	Verb:        "TESTING",
	Subject:     "MLOG",
	Details: []MLogDetailT{
		{"FROM", "UDP_ADDRESS", "STRING"},
		{"FROM", "ID", "STRING"},
		{"NEIGHBORS", "BYTES_TRANSFERRED", "INT"},
	},
}

func BenchmarkSetDetailValues(b *testing.B) {
	vals := []interface{}{"hello", "kitty", 42}
	for i := 0; i < b.N; i++ {
		mlogExampleT.AssignDetails(vals...)
	}
}
