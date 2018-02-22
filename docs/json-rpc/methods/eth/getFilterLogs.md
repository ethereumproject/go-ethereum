
#### eth_getFilterLogs

Returns an array of all logs matching filter with given id.


##### Parameters

1. `QUANTITY` - The filter id.

```js
params: [
"0x16" // 22
]
```

##### Returns

See [eth_getFilterChanges](#eth_getfilterchanges)

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getFilterLogs","params":["0x16"],"id":74}'
```

Result see [eth_getFilterChanges](#eth_getfilterchanges)
