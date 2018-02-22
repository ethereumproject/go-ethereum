
#### db_getHex

Returns binary data from the local database.

**Note** this function is deprecated and will be removed in the future.


##### Parameters

1. `String` - Database name.
2. `String` - Key name.

```js
params: [
"testDB",
"myKey",
]
```

##### Returns

`DATA` - The previously stored data.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"db_getHex","params":["testDB","myKey"],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": "0x68656c6c6f20776f726c64"
}
```
