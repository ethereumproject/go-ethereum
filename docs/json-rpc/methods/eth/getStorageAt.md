
#### eth_getStorageAt

Returns the value from a storage position at a given address.


##### Parameters

1. `DATA`, 20 Bytes - address of the storage.
2. `QUANTITY` - integer of the position in the storage.
3. `QUANTITY|TAG` - integer block number, or the string `"latest"`, `"earliest"` or `"pending"`, see the [default block parameter](#the-default-block-parameter)


```js
params: [
'0x407d73d8a49eeb85d32cf465507dd71d507100c1',
'0x0', // storage position at 0
'0x2' // state at block number 2
]
```

##### Returns

`DATA` - the value at this storage position.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "0x0", "0x2"],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x03"
}
```
