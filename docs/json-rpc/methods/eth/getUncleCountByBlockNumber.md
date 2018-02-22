
#### eth_getUncleCountByBlockNumber

Returns the number of uncles in a block from a block matching the given block number.


##### Parameters

1. `QUANTITY` - integer of a block number, or the string "latest", "earliest" or "pending", see the [default block parameter](#the-default-block-parameter)

```js
params: [
'0xe8', // 232
]
```

##### Returns

`QUANTITY` - integer of the number of uncles in this block.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getUncleCountByBlockNumber","params":["0xe8"],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x1" // 1
}
```
