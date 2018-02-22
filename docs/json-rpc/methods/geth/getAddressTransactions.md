
#### geth_getAddressTransactions

Returns transactions for an address.

Usage requires address-transaction indexes using `geth --atxi` to enable and create indexes during chain sync/import, optionally  using `geth atxi-build` to index pre-existing chain data.


##### Parameters
1. `DATA`, 20 Bytes - address to check for transactions
2. `QUANTITY` - integer block number to filter transactions floor
3. `QUANTITY` - integer block number to filter transactions ceiling
4. `STRING` - `[t|f|tf|]`, use `t` for transactions _to_ the address, `f` for _from_, or `tf`/`''` for both
5. `STRING` - `[s|c|sc|]`, use `s` for _standard_ transactions, `c` for _contracts_, or `sc`/``''` for both
6. `QUANTITY` - integer of index to begin pagination. Using `-1` equivalent to `0`.
7. `QUANTITY` - integer of index to end pagination. Using `-1` equivalent to last transaction _n_.
8. `BOOL` - whether to return transactions in order of oldest first. By default `false` returns transaction hashes ordered by newest transactions first.

```
params: [
'0x407d73d8a49eeb85d32cf465507dd71d507100c1',
123, // earliest block
456, // latest block, use 0 for "undefined", ie. eth.blockNumber
't', // only transactions to this address
'', // both standard and contract transactions
-1, // do not trim transactions for pagination (start)
-1, // do not trim transactions for pagination (end)
false // do not reverse order ('true' will reverse order to be oldest first)
]
```

##### Returns

`Array` - Array of transaction hashes, or an empty array if no transactions found

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"geth_getAddressTransactions","params":["0xb5C694a4cDbc1820Ba4eE8fD6f5AB71a25782534", 5000000, 0, "tf", "sc", -1, -1, false],"id":1}' :8545

// Result
{"jsonrpc":"2.0","id":1,"result":["0xbdaa803ec661db62520eab4aed8854fdea7e04b716849cc67ee7d1c9d94db2d3","0x886e2197a1a703bfed97a39b627f40d8f8beed1fc4814fe8a9618281450f1046","0x4b7f948442732719b31d35139f4269ad021984975c23c35190ac89ef225e95eb","0x35aec85ad9718e937c4e7c11b6f47eebd557cc31b46afc7e19ac888e57e6cdcc","0x0cc2cd8e2b79ef43f441666c0f9de1f06e3690dc3fe64b6fe5d41976115f9184","0x0a06510426a311056e093d1b7a9aabafcb8ce723a6c5c40a9e02824db565844a"]}
```
