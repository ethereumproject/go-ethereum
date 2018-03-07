package eth

import (
	"github.com/ethereumproject/go-ethereum/logger"
	"github.com/ethereumproject/go-ethereum/core/types"
	"github.com/ethereumproject/go-ethereum/common"
	"math/big"
	"github.com/ethereumproject/go-ethereum/rlp"
	"github.com/ethereumproject/go-ethereum/logger/glog"
	"strings"
)

var mlogWwireProtocol = logger.MLogRegisterAvailable("wire", mlogLinesWire)

var mlogLinesWire = []*logger.MLogT{
	mlogWireSendHandshake,
	mlogWireReceiveHandshake,
}

func mlogWireDelegate(p *peer, direction string, msgCode uint64, data interface{}, err error) {
	if !logger.MlogEnabled() {
		return
	}

	var line *logger.MLogT
	var details = []interface{}{
		msgCode,
		ProtocolMessageStringer(uint(msgCode)),
		p.id,
		p.RemoteAddr().String(),
		p.version,
		err,
	}

	// This could obviously be refactored, but I like the explicitness of it.
	switch msgCode {
	case StatusMsg:
		d, ok := data.(*statusData)
		if !ok || d == nil {
			d = &statusData{
				TD: new(big.Int),
				CurrentBlock: common.Hash{},
				GenesisBlock: common.Hash{},
			}
		}
		details = append(details, []interface{}{
			d.ProtocolVersion,
			d.NetworkId,
			d.TD,
			d.CurrentBlock.Hex(),
			d.GenesisBlock.Hex(),
		})
		if direction == "send" {
			line = mlogWireSendHandshake
		} else {
			line = mlogWireReceiveHandshake
		}
	case NewBlockHashesMsg:
		if payload, ok := data.(newBlockHashesData); ok {
			details = append(details,
				len(payload),
				payload[0].Hash,
				payload[0].Number,
			)
		} else {
			glog.Fatal("cant cast: NewBlockHashesMsg", direction)
		}
		if direction == "send" {
			line = mlogWireSendNewBlockHashes
		} else {
			line = mlogWireReceiveNewBlockHashes
		}

	case TxMsg:
		var l int
		if direction == "send" {
			line = mlogWireSendTxs
			if payload, ok := data.(types.Transactions); ok {
				l = payload.Len()
			} else {
				glog.Fatal("cant cast: TxMsg", direction)
			}
		} else {
			line = mlogWireReceiveTxs
			if payload, ok := data.([]*types.Transaction); ok {
				l = len(payload)
			} else {
				glog.Fatal("cant cast: TxMsg", direction)
			}
		}
		details = append(details, l)

	case GetBlockHeadersMsg:
		if payload, ok := data.(*getBlockHeadersData); ok && payload != nil {
			details = append(details, []interface{}{
				payload.Origin.Hash.Hex(),
				payload.Origin.Number,
				payload.Amount,
				payload.Skip,
				payload.Reverse,
			}...)
		 } else {
			glog.Fatal("cant cast: GetBlockHeadersMsg", direction)
		}
		if direction == "send" {
			line = mlogWireSendGetBlockHeaders
		} else {
			line = mlogWireReceiveGetBlockHeaders
		}

	case BlockHeadersMsg:
		if payload, ok := data.([]*types.Header); ok {
			details = append(details,
				len(payload),
			)
			if len(payload) > 0 && payload[0] != nil {
				details = append(details,
					payload[0].Hash().Hex(),
					payload[0].Number.Uint64(),
				)
			} else {
				details = append(details,
					common.Hash{}.Hex(),
					-1,
				)
			}
		} else {
			glog.Fatal("cant cast: BlockHeadersMsg", direction)
		}
		if direction == "send" {
			line = mlogWireSendBlockHeaders
		} else {
			line = mlogWireReceiveBlockHeaders
		}

	case GetBlockBodiesMsg:
		if direction == "send" {
			line = mlogWireSendGetBlockBodies
			if payload, ok := data.([]common.Hash); ok {
				details = append(details, len(payload))
				if len(payload) > 0 {
					details = append(details, payload[0].Hex())
				} else {
					details = append(details, common.Hash{}.Hex())
				}
			} else {
				glog.Fatal("cant cast: GetBlockBodiesMsg", direction)
			}
		} else {
			line = mlogWireReceiveGetBlockBodies
			if payload, ok := data.([]rlp.RawValue); ok {
				details = append(details,
					len(payload),
				)
			} else {
				glog.Fatal("cant cast: GetBlockBodiesMsg", direction)
			}
		}

	case BlockBodiesMsg:
		if direction == "send" {
			line = mlogWireSendBlockBodies
			if payload, ok := data.([]rlp.RawValue); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: BlockBodiesMsg", direction)
			}
		} else {
			line = mlogWireReceiveBlockBodies
			if payload, ok := data.(blockBodiesData); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: BlockBodiesMsg", direction)
			}
		}

	case NewBlockMsg:
		if payload, ok := data.(newBlockData); ok {
			if b := payload.Block; b != nil {
				details = append(details,
					b.Hash().Hex(),
					b.Number().Uint64(),
				)
			} else {
				details = append(details,
					common.Hash{}.Hex(),
					-1,
				)
			}
			if td := payload.TD; td != nil {
				details = append(details, td)
			} else {
				details = append(details, 0)
			}
		} else {
			glog.Fatal("cant cast: NewBlockMsg", direction)
		}
		if direction == "send" {
			line = mlogWireSendNewBlock
		} else {
			line = mlogWireReceiveNewBlock
		}

	case GetNodeDataMsg:
		if direction == "send" {
			line = mlogWireSendGetNodeData
			if payload, ok := data.([]common.Hash); ok {
				details = append(details, len(payload))
				if len(payload) > 0 {
					details =append(details, payload[0].Hex())
				} else {
					details =append(details, common.Hash{}.Hex())
				}
			} else {
				glog.Fatal("cant cast: GetNodeDataMsg", direction)
			}
		} else {
			line = mlogWireReceiveGetNodeData
			if payload, ok := data.([][]byte); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: GetNodeDataMsg", direction)
			}
		}

	case NodeDataMsg:
		if payload, ok := data.([][]byte); ok {
			details = append(details, len(payload))
		} else {
			glog.Fatal("cant cast: GetNNodeDataMsg", direction)
		}
		if direction == "send" {
			line = mlogWireSendNodeData
		} else {
			line = mlogWireReceiveNodeData
		}

	case GetReceiptsMsg:
		if direction == "send" {
			line = mlogWireSendGetReceipts
			if payload, ok := data.([]common.Hash); ok {
				details = append(details, len(payload))
				if len(payload) > 0 {
					details = append(details, payload[0].Hex())
				} else {
					details = append(details, common.Hash{}.Hex())
				}
			} else {
				glog.Fatal("cant cast: GetReceiptsMsg", direction)
			}
		} else {
			line = mlogWireReceiveGetReceipts
			if payload, ok := data.([]rlp.RawValue); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: GetNNodGetReceiptsMsg", direction)
			}
		}

	case ReceiptsMsg:
		if direction == "send" {
			line = mlogWireSendReceipts
			if payload, ok := data.([]rlp.RawValue); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: ReceiptsMsg", direction)
			}
		} else {
			line = mlogWireReceiveReceipts
			if payload, ok := data.([][]*types.Receipt); ok {
				details = append(details, len(payload))
			} else {
				glog.Fatal("cant cast: ReceiptsMsg", direction)
			}
		}

	default:
		line = mlogWireReceiveInvalid
	}

	if line == nil {
		glog.Fatalln("log line cannot be nil", p, direction, ProtocolMessageStringer(uint(msgCode)))
	}

	line.AssignDetails(
		details...
	).Send(mlogWwireProtocol)
}

var mlogWireCommonDetails = []logger.MLogDetailT{
	{Owner: "WIRE", Key: "CODE", Value: "INT"},
	{Owner: "WIRE", Key: "NAME", Value: "STRING"},
	{Owner: "WIRE", Key: "REMOTE_ID", Value: "STRING"},
	{Owner: "WIRE", Key: "REMOTE_ADDR", Value: "STRING"},
	{Owner: "WIRE", Key: "REMOTE_VERSION", Value: "INT"},
	{Owner: "WIRE", Key: "ERROR", Value: "STRING_OR_NULL"},
}

var mlogWireSendHandshake = &logger.MLogT{
	Description: "Called once for each outgoing StatusMsg (handshake) event.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(StatusMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "PROTOCOL_VERSION", Value: "INT"},
		{Owner: "MSG", Key: "NETWORK_ID", Value: "INT"},
		{Owner: "MSG", Key: "TD", Value: "BIGINT"},
		{Owner: "MSG", Key: "HEAD_BLOCK_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "GENESIS_BLOCK_HASH", Value: "STRING"},
	}...),
}

var mlogWireReceiveHandshake = &logger.MLogT{
	Description: "Called once for each incoming StatusMsg (handshake) event.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(StatusMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "PROTOCOL_VERSION", Value: "INT"},
		{Owner: "MSG", Key: "NETWORK_ID", Value: "INT"},
		{Owner: "MSG", Key: "TD", Value: "BIGINT"},
		{Owner: "MSG", Key: "HEAD_BLOCK_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "GENESIS_BLOCK_HASH", Value: "STRING"},
	}...),
}

var mlogWireSendNewBlockHashes = &logger.MLogT{
	Description: "Called once for each outgoing SendNewBlockHashes message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(NewBlockHashesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "FIRST_NUMBER", Value: "INT"},
	}...),
}

var mlogWireReceiveNewBlockHashes = &logger.MLogT{
	Description: "Called once for each incoming SendNewBlockHashes message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(NewBlockHashesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "FIRST_NUMBER", Value: "INT"},
	}...),
}

var mlogWireSendTxs = &logger.MLogT{
	Description: "Called once for each outgoing SendNewBTxMsglockHashes message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(TxMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireReceiveTxs = &logger.MLogT{
	Description: "Called once for each incoming TxMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(TxMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireSendGetBlockHeaders = &logger.MLogT{
	Description: "Called once for each outgoing GetBlockHeadersMsg message. Note that origin value will be EITHER hash or origin.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetBlockHeadersMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "ORIGIN_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "ORIGIN_NUMBER", Value: "INT"},
		{Owner: "MSG", Key: "AMOUNT", Value: "INT"},
		{Owner: "MSG", Key: "SKIP", Value: "INT"},
		{Owner: "MSG", Key: "REVERSE", Value: "BOOL"},
	}...),
}

var mlogWireReceiveGetBlockHeaders = &logger.MLogT{
	Description: "Called once for each incoming GetBlockHeadersMsg message. Note that origin value will be EITHER hash or origin.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetBlockHeadersMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "ORIGIN_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "ORIGIN_NUMBER", Value: "INT"},
		{Owner: "MSG", Key: "AMOUNT", Value: "INT"},
		{Owner: "MSG", Key: "SKIP", Value: "INT"},
		{Owner: "MSG", Key: "REVERSE", Value: "BOOL"},
	}...),
}

var mlogWireSendBlockHeaders = &logger.MLogT{
	Description: "Called once for each outgoing BlockHeadersMsg message. Note that origin value will be EITHER hash or origin.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(BlockHeadersMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "FIRST_NUMBER", Value: "INT"},
	}...),
}

var mlogWireReceiveBlockHeaders = &logger.MLogT{
	Description: "Called once for each incoming BlockHeadersMsg message. Note that origin value will be EITHER hash or origin.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(BlockHeadersMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "FIRST_NUMBER", Value: "INT"},
	}...),
}

var mlogWireSendGetBlockBodies = &logger.MLogT{
	Description: "Called once for each outgoing GetBlockBodiesMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetBlockBodiesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
	}...),
}

var mlogWireReceiveGetBlockBodies = &logger.MLogT{
	Description: "Called once for each incoming GetBlockBodiesMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetBlockBodiesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireSendBlockBodies = &logger.MLogT{
	Description: "Called once for each outgoing BlockBodiesMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(BlockBodiesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireReceiveBlockBodies = &logger.MLogT{
	Description: "Called once for each incoming BlockBodiesMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(BlockBodiesMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireSendNewBlock = &logger.MLogT{
	Description: "Called once for each outgoing NewBlockMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(NewBlockMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "BLOCK_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "BLOCK_NUMBER", Value: "INT"},
		{Owner: "MSG", Key: "TD", Value: "BIGINT"},
	}...),
}

var mlogWireReceiveNewBlock = &logger.MLogT{
	Description: "Called once for each incoming NewBlockMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(NewBlockMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "BLOCK_HASH", Value: "STRING"},
		{Owner: "MSG", Key: "BLOCK_NUMBER", Value: "INT"},
		{Owner: "MSG", Key: "TD", Value: "BIGINT"},
	}...),
}

var mlogWireSendGetNodeData = &logger.MLogT{
	Description: "Called once for each outgoing GetNodeDataMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetNodeDataMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
	}...),
}

var mlogWireReceiveGetNodeData = &logger.MLogT{
	Description: "Called once for each incoming GetNodeDataMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetNodeDataMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireSendNodeData = &logger.MLogT{
	Description: "Called once for each outgoing NodeDataMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(NodeDataMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireReceiveNodeData = &logger.MLogT{
	Description: "Called once for each incoming NodeDataMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(NodeDataMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireSendGetReceipts = &logger.MLogT{
	Description: "Called once for each outgoing GetReceiptsMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetReceiptsMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
		{Owner: "MSG", Key: "FIRST_HASH", Value: "STRING"},
	}...),
}

var mlogWireReceiveGetReceipts = &logger.MLogT{
	Description: "Called once for each incoming GetReceiptsMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(GetReceiptsMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "STRING"},
	}...),
}

var mlogWireSendReceipts = &logger.MLogT{
	Description: "Called once for each outgoing ReceiptsMsg message.",
	Receiver: "WIRE",
	Verb: "SEND",
	Subject: strings.ToUpper(ProtocolMessageStringer(ReceiptsMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "INT"},
	}...),
}

var mlogWireReceiveReceipts = &logger.MLogT{
	Description: "Called once for each incoming ReceiptsMsg message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: strings.ToUpper(ProtocolMessageStringer(ReceiptsMsg)),
	Details: append(mlogWireCommonDetails, []logger.MLogDetailT{
		{Owner: "MSG", Key: "LEN_ITEMS", Value: "STRING"},
	}...),
}

var mlogWireReceiveInvalid = &logger.MLogT{
	Description: "Called once for each incoming INVALID message.",
	Receiver: "WIRE",
	Verb: "RECEIVE",
	Subject: "INVALID",
	Details: mlogWireCommonDetails,
}