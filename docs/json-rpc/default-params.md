## Default block parameters

The following methods have a extra default block parameter:

- [eth_getBalance](./methods/eth/getBalance.md)
- [eth_getCode](./methods/eth/getCode.md)
- [eth_getTransactionCount](./methods/eth/getTransactionCount.md)
- [eth_getStorageAt](./methods/eth/getStorageAt.md)
- [eth_call](./methods/eth/call.md)

When requests are made that act on the state of ethereum, the last default block parameter determines the height of the block.

The following options are possible for the defaultBlock parameter:

- `HEX String` - an integer block number
- `String "earliest"` for the earliest/genesis block
- `String "latest"` - for the latest mined block
- `String "pending"` - for the pending state/transactions
