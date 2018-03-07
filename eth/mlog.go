package eth

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/common"
	"math/big"
	"github.com/ethereumproject/go-ethereum/rlp"
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
			d = &statusData{}
			d.TD = new(big.Int)
			d.CurrentBlock = common.Hash{}
			d.GenesisBlock = common.Hash{}
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
	var eventData interface{}

	// This could obviously be refactored, but I like the explicitness of it.
	switch msgcode {
	case NewBlockHashesMsg:
		if payload, ok := data.(newBlockHashesData); ok {
			payloadLength = len(payload)
		} else {
			glog.Fatal("cant cast: NewBlockHashesMsg", direction)
		}
	case TxMsg:
		if payload, ok := data.(types.Transactions); ok {
			payloadLength = payload.Len()
		} else {
			glog.Fatal("cant cast: TxMsg", direction)
		}
	case GetBlockHeadersMsg:
		if payload, ok := data.(*getBlockHeadersData); ok {
			eventData = payload
		 } else {
			glog.Fatal("cant cast: GetBlockHeadersMsg", direction)
		}
	case BlockHeadersMsg:
		if payload, ok := data.([]*types.Header); ok {
			payloadLength = len(payload)
		} else {
			glog.Fatal("cant cast: BlockHeadersMsg", direction)
		}
	case GetBlockBodiesMsg:
		if direction == "send" {
			if payload, ok := data.([]common.Hash); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetBlockBodiesMsg", direction)
			}
		} else {
			if payload, ok := data.([]rlp.RawValue); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetBlockBodiesMsg", direction)
			}
		}
	case BlockBodiesMsg:
		if direction == "send" {
			if payload, ok := data.([]rlp.RawValue); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: BlockBodiesMsg", direction)
			}
		} else {
			if payload, ok := data.(blockBodiesData); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: BlockBodiesMsg", direction)
			}
		}
	case NewBlockMsg:
		if payload, ok := data.(newBlockData); ok {
			eventData = payload
		} else {
			glog.Fatal("cant cast: NewBlockMsg", direction)
		}
		payloadLength = 1
	case GetNodeDataMsg:
		if direction == "send" {
			if payload, ok := data.([]common.Hash); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetNodeDataMsg", direction)
			}
		} else {
			if payload, ok := data.([][]byte); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetNodeDataMsg", direction)
			}
		}
	case NodeDataMsg:
		if payload, ok := data.([][]byte); ok {
			payloadLength = len(payload)
		} else {
			glog.Fatal("cant cast: GetNNodeDataMsg", direction)
		}
	case GetReceiptsMsg:
		if direction == "send" {
			if payload, ok := data.([]common.Hash); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetReceiptsMsg", direction)
			}
		} else {
			if payload, ok := data.([]rlp.RawValue); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: GetNNodGetReceiptsMsg", direction)
			}
		}
	case ReceiptsMsg:
		//[][]*types.Receipt
		if direction == "send" {
			if payload, ok := data.([]rlp.RawValue); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: ReceiptsMsg", direction)
			}
		} else {
			if payload, ok := data.([][]*types.Receipt); ok {
				payloadLength = len(payload)
			} else {
				glog.Fatal("cant cast: ReceiptsMsg", direction)
			}
		}
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
		eventData, // can be nil
		err,       // can be nil
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
		{Owner: "TRANSFER", Key: "DATA", Value: "OBJECT"}, // only present for GetHeadersData messages
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
		{Owner: "TRANSFER", Key: "DATA", Value: "OBJECT"},
		{Owner: "TRANSFER", Key: "ERROR", Value: "STRING_OR_NULL"},
	},
}


