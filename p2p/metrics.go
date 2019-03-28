// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"net"

	"github.com/ethereumproject/go-ethereum/metrics"
)

// meteredConn wraps a network TCP connection for metrics.
type meteredConn struct {
	net.Conn
	markBytes func(int64)
}

func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	if ingress {
		metrics.P2PIn.Mark(1)
		return &meteredConn{conn, metrics.P2PInBytes.Mark}
	} else {
		metrics.P2POut.Mark(1)
		return &meteredConn{conn, metrics.P2POutBytes.Mark}
	}
}

func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	c.markBytes(int64(n))
	return
}

func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	c.markBytes(int64(n))
	return
}
