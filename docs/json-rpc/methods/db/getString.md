
#### db_getString

Returns string from the local database.

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

`String` - The previously stored string.


##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"db_getString","params":["testDB","myKey"],"id":73}'

// Result
{
"id":1,
"jsonrpc":"2.0",
"result": "myString"
}
```
