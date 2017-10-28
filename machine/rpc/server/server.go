// Copyright (c) 2017 ETCDEV Team

// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package server

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"github.com/ethereumproject/go-ethereum/machine/classic"
	"github.com/ethereumproject/go-ethereum/machine/rpc/client"
	"github.com/ethereumproject/go-ethereum/rpc"
	"net"
)

type Server struct {
	handler *rpc.Server
	lstnr   net.Listener
	stop    bool
	cStop   chan struct{}
}

func NewServer() *Server {
	handler := rpc.NewServer()
	svc := &MachineSvc{Machine: classic.NewMachine()}
	if err := handler.RegisterName("vm", svc); err != nil {
		panic(err) // should not occur, but for sure
	}
	return &Server{handler, nil, false, make(chan struct{})}
}

func (self *Server) ServeConnection(conn net.Conn) {
	for !self.stop {
		glog.V(logger.Debug).Infof("VM SERVER is waiting for a new request\n")
		err := self.handler.ServeSingleRequest(rpc.NewJSONCodec(conn), rpc.OptionMethodInvocation)
		if err != nil {
			break
		}
		glog.V(logger.Debug).Infof("VM SERVER is done on the last request\n")
	}
}

func (self *Server) ServeIpc(name string) error {
	lstnr, err := rpc.IpcListen(client.IpcNameFrom(name))
	if err != nil {
		return err
	}
	self.lstnr = lstnr
	go func() {
		glog.V(logger.Debug).Infof("VM SERVER is on duty")
		for {
			glog.V(logger.Debug).Infof("VM SERVER is wating for a new client")
			conn, err := self.lstnr.Accept()
			if err != nil {
				if err.Error() != "use of closed network connection" {
					glog.Error(err)
				}
				break
			} else {
				glog.V(logger.Debug).Infof("VM SERVER accepted connection from %v\n", conn.RemoteAddr())
				self.ServeConnection(conn)
				glog.V(logger.Debug).Infof("VM SERVER is done on connection from %v\n", conn.RemoteAddr())
			}
		}
		glog.V(logger.Debug).Infof("VM SERVER is stopped")
		close(self.cStop)
	}()
	return nil
}

func (self *Server) Close() {
	if self.cStop != nil && self.lstnr != nil {
		self.lstnr.Close()
		<-self.cStop
		self.cStop = nil
		self.lstnr = nil
	}
}
