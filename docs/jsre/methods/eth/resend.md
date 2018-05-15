
#### eth.resend

eth.resend(tx, <optional gas price>, <optional gas limit>)

Resends the given transaction returned by `pendingTransactions()` and allows you to overwrite the gas price and gas limit of the transaction.

##### Example

```javascript
eth.sendTransaction({from: eth.accounts[0], to: "...", gasPrice: "1000"})
var tx = eth.pendingTransactions[0]
eth.resend(tx, web3.toWei(10, "szabo"))
```
