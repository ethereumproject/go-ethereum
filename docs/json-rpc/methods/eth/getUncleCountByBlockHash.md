
#### eth_getUncleCountByBlockHash

Returns the number of uncles in a block from a block matching the given block hash.


##### Parameters

1. `DATA`, 32 Bytes - hash of a block

```js
params: [
'0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238'
]
```

##### Returns

`QUANTITY` - integer of the number of uncles in this block.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getUncleCountByBlockHash","params":["0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"],"id"Block:1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x1" // 1
}
```
