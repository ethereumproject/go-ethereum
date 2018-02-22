
#### eth_getBalance

Returns the balance of the account of given address.

##### Parameters

1. `DATA`, 20 Bytes - address to check for balance.
2. `QUANTITY|TAG` - integer block number, or the string `"latest"`, `"earliest"` or `"pending"`, see the [default block parameter](#the-default-block-parameter)

```js
params: [
'0x407d73d8a49eeb85d32cf465507dd71d507100c1',
'latest'
]
```

##### Returns

`QUANTITY` - integer of the current balance in wei.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x407d73d8a49eeb85d32cf465507dd71d507100c1", "latest"],"id":1}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": "0x0234c8a3397aab58" // 158972490234375000
}
```
