
#### eth_uninstallFilter

Uninstalls a filter with given id. Should always be called when watch is no longer needed.
Additonally Filters timeout when they aren't requested with [eth_getFilterChanges](#eth-getfilterchanges) for a period of time.


##### Parameters

1. `QUANTITY` - The filter id.

```js
params: [
"0xb" // 11
]
```

##### Returns

`Boolean` - `true` if the filter was successfully uninstalled, otherwise `false`.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_uninstallFilter","params":["0xb"],"id":73}'

// Result
{
"id":1,
"jsonrpc": "2.0",
"result": true
}
```
