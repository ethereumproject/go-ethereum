
#### eth_getUncleByBlockHashAndIndex

Returns information about a uncle of a block by hash and uncle index position.


##### Parameters


1. `DATA`, 32 Bytes - hash a block.
2. `QUANTITY` - the uncle's index position.

```js
params: [
'0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b',
'0x0' // 0
]
```

##### Returns

See [eth_getBlockByHash](#eth-getblockbyhash)

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getUncleByBlockHashAndIndex","params":["0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "0x0"],"id":1}'
```

Result see [eth_getBlockByHash](#eth-getblockbyhash)

**Note**: An uncle doesn't contain individual transactions.
