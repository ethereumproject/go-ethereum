
#### admin.startWS

admin.startWS(host, portNumber, allowedOrigins, modules)

Starts the websocket server for the [JSON-RPC](https://github.com/ethereumproject/wiki/wiki/JSON-RPC).

##### Returns

`true` on success, otherwise `false`.

##### Example

```javascript
admin.startWS("127.0.0.1", 8546, "*", "web3,net,eth")
// true
```
