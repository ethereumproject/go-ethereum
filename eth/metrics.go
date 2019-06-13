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

package eth

import (
	"github.com/ethereumproject/go-ethereum/metrics"
	"github.com/ethereumproject/go-ethereum/p2p"
)

// meteredMsgReadWriter is a wrapper around a p2p.MsgReadWriter, capable of
// accumulating the above defined metrics based on the data stream contents.
type meteredMsgReadWriter struct {
	p2p.MsgReadWriter     // Wrapped message stream to meter
	version           int // Protocol version to select correct meters
}

// newMeteredMsgWriter wraps a p2p MsgReadWriter with metering support. If the
// metrics system is disabled, this function returns the original object.
func newMeteredMsgWriter(rw p2p.MsgReadWriter) p2p.MsgReadWriter {
	return &meteredMsgReadWriter{MsgReadWriter: rw}
}

// Init sets the protocol version used by the stream to know which meters to
// increment in case of overlapping message ids between protocol versions.
func (rw *meteredMsgReadWriter) Init(version int) {
	rw.version = version
}

func (rw *meteredMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	msg, err := rw.MsgReadWriter.ReadMsg()
	if err != nil {
		return msg, err
	}

	messages, bytes := metrics.MsgMiscIn, metrics.MsgMiscInBytes
	switch {
	case msg.Code == BlockHeadersMsg:
		messages, bytes = metrics.MsgHeaderIn, metrics.MsgHeaderInBytes
	case msg.Code == BlockBodiesMsg:
		messages, bytes = metrics.MsgBodyIn, metrics.MsgBodyInBytes
	case rw.version >= eth63 && msg.Code == NodeDataMsg:
		messages, bytes = metrics.MsgStateIn, metrics.MsgStateInBytes
	case rw.version >= eth63 && msg.Code == ReceiptsMsg:
		messages, bytes = metrics.MsgReceiptIn, metrics.MsgReceiptInBytes
	case msg.Code == NewBlockHashesMsg:
		messages, bytes = metrics.MsgHashIn, metrics.MsgHashInBytes
	case msg.Code == NewBlockMsg:
		messages, bytes = metrics.MsgBlockIn, metrics.MsgBlockInBytes
	case msg.Code == TxMsg:
		messages, bytes = metrics.MsgTXNIn, metrics.MsgTXNInBytes
	}
	messages.Mark(1)
	bytes.Mark(int64(msg.Size))

	return msg, nil
}

func (rw *meteredMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	messages, bytes := metrics.MsgMiscOut, metrics.MsgMiscOutBytes
	switch {
	case msg.Code == BlockHeadersMsg:
		messages, bytes = metrics.MsgHeaderOut, metrics.MsgHeaderOutBytes
	case msg.Code == BlockBodiesMsg:
		messages, bytes = metrics.MsgBodyOut, metrics.MsgBodyOutBytes
	case rw.version >= eth63 && msg.Code == NodeDataMsg:
		messages, bytes = metrics.MsgStateOut, metrics.MsgStateOutBytes
	case rw.version >= eth63 && msg.Code == ReceiptsMsg:
		messages, bytes = metrics.MsgReceiptOut, metrics.MsgReceiptOutBytes
	case msg.Code == NewBlockHashesMsg:
		messages, bytes = metrics.MsgHashOut, metrics.MsgHashOutBytes
	case msg.Code == NewBlockMsg:
		messages, bytes = metrics.MsgBlockOut, metrics.MsgBlockOutBytes
	case msg.Code == TxMsg:
		messages, bytes = metrics.MsgTXNOut, metrics.MsgTXNOutBytes
	}
	messages.Mark(1)
	bytes.Mark(int64(msg.Size))

	return rw.MsgReadWriter.WriteMsg(msg)
}
