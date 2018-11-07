// Copyright (c) 2017 Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package dscp_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/aristanetworks/goarista/dscp"
)

func TestDialTCPWithTOS(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer listen.Close()

	done := make(chan struct{})
	go func() {
		conn, err := listen.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		buf := []byte{'!'}
		conn.Write(buf)
		n, err := conn.Read(buf)
		if n != 1 || err != nil {
			t.Fatalf("Read returned %d / %s", n, err)
		} else if buf[0] != '!' {
			t.Fatalf("Expected to read '!' but got %q", buf)
		}
		close(done)
	}()
	conn, err := dscp.DialTCPWithTOS(nil, listen.Addr().(*net.TCPAddr), 40)
	if err != nil {
		t.Fatal("Connection failed:", err)
	}
	defer conn.Close()
	buf := make([]byte, 1)
	n, err := conn.Read(buf)
	if n != 1 || err != nil {
		t.Fatalf("Read returned %d / %s", n, err)
	} else if buf[0] != '!' {
		t.Fatalf("Expected to read '!' but got %q", buf)
	}
	conn.Write(buf)
	<-done
}

func TestDialTCPTimeoutWithTOS(t *testing.T) {
	raddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	for name, td := range map[string]*net.TCPAddr{
		"ipNoPort": &net.TCPAddr{
			IP: net.ParseIP("127.0.0.42"), Port: 0,
		},
		"ipWithPort": &net.TCPAddr{
			IP: net.ParseIP("127.0.0.42"), Port: 10001,
		},
	} {
		t.Run(name, func(t *testing.T) {
			l, err := net.ListenTCP("tcp", raddr)
			if err != nil {
				t.Fatal(err)
			}
			defer l.Close()

			var srcAddr net.Addr
			done := make(chan struct{})
			go func() {
				conn, err := l.Accept()
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				srcAddr = conn.RemoteAddr()
				close(done)
			}()

			conn, err := dscp.DialTCPTimeoutWithTOS(td, l.Addr().(*net.TCPAddr), 40, 5*time.Second)
			if err != nil {
				t.Fatal("Connection failed:", err)
			}
			defer conn.Close()

			pfx := td.IP.String() + ":"
			if td.Port > 0 {
				pfx = fmt.Sprintf("%s%d", pfx, td.Port)
			}
			<-done
			if !strings.HasPrefix(srcAddr.String(), pfx) {
				t.Fatalf("DialTCPTimeoutWithTOS wrong address: %q instead of %q", srcAddr, pfx)
			}
		})
	}
}
