
#### shh_getMessages

Get all messages matching a filter, which are still existing in the node's buffer.

**Note** calling this method, will also reset the buffer for the [shh_getFilterChanges](#shh_getfilterchanges) method, so that you won't receive duplicate messages.

##### Parameters

1. `QUANTITY` - The filter id.

```js
params: [
"0x7" // 7
]
```

##### Returns

See [shh_getFilterChanges](#shh_getfilterchanges)

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"shh_getMessages","params":["0x7"],"id":73}'
```

Result see [shh_getFilterChanges](#shh_getfilterchanges)
