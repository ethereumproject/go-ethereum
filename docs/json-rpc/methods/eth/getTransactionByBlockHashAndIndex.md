
#### eth_getTransactionByBlockHashAndIndex

Returns information about a transaction by block hash and transaction index position.


##### Parameters

1. `DATA`, 32 Bytes - hash of a block.
2. `QUANTITY` - integer of the transaction index position.

```js
params: [
'0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331',
'0x0' // 0
]
```

##### Returns

See [eth_getBlockByHash](#eth-gettransactionbyhash)

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getTransactionByBlockHashAndIndex","params":[0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b, "0x0"],"id":1}'
```

Result see [eth_getTransactionByHash](#eth-gettransactionbyhash)
