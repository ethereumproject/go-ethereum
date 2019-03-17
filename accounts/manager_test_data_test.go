package accounts

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"
)

var testSigData = make([]byte, 32)

func tmpManager(t *testing.T) (string, *Manager) {
	rand.Seed(time.Now().UnixNano())
	dir, err := ioutil.TempDir("", fmt.Sprintf("eth-manager-mem-test-%d-%d", os.Getpid(), rand.Int()))
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(dir, veryLightScryptN, veryLightScryptP, false)
	if err != nil {
		t.Fatal(err)
	}
	return dir, m
}
