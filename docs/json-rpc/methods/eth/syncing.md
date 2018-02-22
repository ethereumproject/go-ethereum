
#### eth_syncing

Returns an object object with data about the sync status or FALSE.


##### Parameters
none

##### Returns

`Object|Boolean`, An object with sync status data or `FALSE`, when not syncing:
- `startingBlock`: `QUANTITY` - The block at which the import started (will only be reset, after the sync reached his head)
- `currentBlock`: `QUANTITY` - The current block, same as eth_blockNumber
- `highestBlock`: `QUANTITY` - The estimated highest block

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_isSyncing","params":[],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": {
startingBlock: '0x384',
currentBlock: '0x386',
highestBlock: '0x454'
}
}
// Or when not syncing
{
"id":1,
"jsonrpc": "2.0",
"result": false
}
```
