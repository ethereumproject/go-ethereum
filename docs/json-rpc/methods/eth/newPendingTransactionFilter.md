
#### eth_newPendingTransactionFilter

Creates a filter in the node, to notify when new pending transactions arrive.
To check if the state has changed, call [eth_getFilterChanges](#eth-getfilterchanges).

##### Parameters
None

##### Returns

`QUANTITY` - A filter id.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_newPendingTransactionFilter","params":[],"id":73}'

// Result
{
"id":1,
"jsonrpc":  "2.0",
"result": "0x1" // 1
}
```
