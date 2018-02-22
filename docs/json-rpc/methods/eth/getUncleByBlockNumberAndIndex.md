
#### eth_getUncleByBlockNumberAndIndex

Returns information about a uncle of a block by number and uncle index position.


##### Parameters

1. `QUANTITY|TAG` - a block number, or the string `"earliest"`, `"latest"` or `"pending"`, as in the [default block parameter](#the-default-block-parameter).
2. `QUANTITY` - the uncle's index position.

```js
params: [
'0x29c', // 668
'0x0' // 0
]
```

##### Returns

See [eth_getBlockByHash](#eth-getblockbyhash)

**Note**: An uncle doesn't contain individual transactions.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getUncleByBlockNumberAndIndex","params":["0x29c", "0x0"],"id":1}'
```

Result see [eth_getBlockByHash](#eth-getblockbyhash)
