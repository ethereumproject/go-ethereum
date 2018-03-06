package eth

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/logger/glog"
)

var mlogWwireProtocol = logger.MLogRegisterAvailable("wire", mlogLinesWire)

var mlogLinesWire = []*logger.MLogT{
	mlogWireReceiveTransfer,
	mlogWireSendTransfer,
	mlogWireSendHandshake,
	mlogWireReceiveHandshake,
}

type arbitrarySlice []interface{}
func mustGetLenIfPossible(data interface{}) int {
	s, ok := data.(arbitrarySlice)
	if !ok {
		return 0
	}
	return len(s)
}


func mlogWireDelegate(p *peer, direction string, msgcode uint64, data interface{}, err error) {
	if !logger.MlogEnabled() {
		return
	}

	if msgcode == StatusMsg {
		d, ok := data.(*statusData)
		if !ok {
			glog.Fatal("yikes. FIXME")
		}

		logLine := mlogWireReceiveHandshake
		if direction == "send" {
			logLine = mlogWireSendHandshake
		}
		logLine.AssignDetails(
			msgcode,
			ProtocolMessageStringer(uint(msgcode)),
			p.id,
			p.RemoteAddr().String(),
			p.version,
			d.ProtocolVersion,
			d.NetworkId,
			d.TD,
			d.CurrentBlock.Hex(),
			d.GenesisBlock.Hex(),
			err,
		).Send(mlogWwireProtocol)

		return
	}

	// set length of hashes if relevant
	var payloadLength int

	// This could obviously be refactored, but I like the explicitness of it.
	switch msgcode {
	case NewBlockHashesMsg:
		payloadLength = mustGetLenIfPossible(data)
	case TxMsg:
		payloadLength = mustGetLenIfPossible(data)
	case GetBlockHeadersMsg:
	case BlockHeadersMsg:
		payloadLength = mustGetLenIfPossible(data)
	case GetBlockBodiesMsg:
	case BlockBodiesMsg:
		payloadLength = mustGetLenIfPossible(data)
	case NewBlockMsg:
		payloadLength = 1
	case GetNodeDataMsg:
		payloadLength = mustGetLenIfPossible(data)
	case NodeDataMsg:
		payloadLength = mustGetLenIfPossible(data)
	case GetReceiptsMsg:
		payloadLength = mustGetLenIfPossible(data)
	case ReceiptsMsg:
		payloadLength = mustGetLenIfPossible(data)
	default:
	}

	logLine := mlogWireReceiveTransfer
	if direction == "send" {
		logLine = mlogWireSendTransfer
	}

	logLine.AssignDetails(
		msgcode,
		ProtocolMessageStringer(uint(msgcode)),
		p.id,
		p.RemoteAddr().String(),
		p.version,
		payloadLength,
		err,
	).Send(mlogWwireProtocol)

}

var mlogWireSendHandshake = &logger.MLogT{
	Description: "Called once for each outgoing Wire Protocol handshake event.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: "HANDSHAKE",
	Details: []logger.MLogDetailT{
		{Owner: "WIRE", Key: "CODE", Value: "INT"},
		{Owner: "WIRE", Key: "NAME", Value: "STRING"},
		//{Owner: "HANDSHAKE", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "WIRE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_VERSION", Value: "INT"},
		{Owner: "HANDSHAKE", Key: "PROTOCOL_VERSION", Value: "INT"},
		{Owner: "HANDSHAKE", Key: "NETWORK_ID", Value: "INT"},
		{Owner: "HANDSHAKE", Key: "TD", Value: "BIGINT"},
		{Owner: "HANDSHAKE", Key: "HEAD_BLOCK_HASH", Value: "STRING"},
		{Owner: "HANDSHAKE", Key: "GENESIS_BLOCK_HASH", Value: "STRING"},
		{Owner: "HANDSHAKE", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogWireReceiveHandshake = &logger.MLogT{
	Description: "Called once for each incoming Wire Protocol handshake event.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: "HANDSHAKE",
	Details: []logger.MLogDetailT{
		{Owner: "WIRE", Key: "CODE", Value: "INT"},
		{Owner: "WIRE", Key: "NAME", Value: "STRING"},
		//{Owner: "HANDSHAKE", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "WIRE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_VERSION", Value: "STRING"},
		{Owner: "HANDSHAKE", Key: "PROTOCOL_VERSION", Value: "INT"},
		{Owner: "HANDSHAKE", Key: "NETWORK_ID", Value: "INT"},
		{Owner: "HANDSHAKE", Key: "TD", Value: "BIGINT"},
		{Owner: "HANDSHAKE", Key: "HEAD_BLOCK_HASH", Value: "STRING"},
		{Owner: "HANDSHAKE", Key: "GENESIS_BLOCK_HASH", Value: "STRING"},
		{Owner: "HANDSHAKE", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogWireSendTransfer = &logger.MLogT{
	Description: "Called once for each outgoing Wire Protocol transfer event.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: "TRANSFER",
	Details: []logger.MLogDetailT{
		{Owner: "WIRE", Key: "CODE", Value: "INT"},
		{Owner: "WIRE", Key: "NAME", Value: "STRING"},
		//{Owner: "TRANSFER", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "WIRE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_VERSION", Value: "STRING"},
		{Owner: "TRANSFER", Key: "PAYLOAD_LENGTH", Value: "INT"}, // eg length of hashes
		{Owner: "TRANSFER", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}

var mlogWireReceiveTransfer = &logger.MLogT{
	Description: "Called once for each incoming Wire Protocol transfer event.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: "TRANSFER",
	Details: []logger.MLogDetailT{
		{Owner: "WIRE", Key: "CODE", Value: "INT"},
		{Owner: "WIRE", Key: "NAME", Value: "STRING"},
		//{Owner: "TRANSFER", Key: "SIZE", Value: "INT"}, // size in bytes
		{Owner: "WIRE", Key: "REMOTE_ID", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_ADDR", Value: "STRING"},
		{Owner: "WIRE", Key: "REMOTE_VERSION", Value: "STRING"},
		{Owner: "TRANSFER", Key: "PAYLOAD_LENGTH", Value: "INT"},
		{Owner: "TRANSFER", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}


