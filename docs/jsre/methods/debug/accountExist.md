
#### debug.accountExist

```
debug.accountExist(address, blockNumber)
```

##### Returns

Returns `BOOL` if a given account exists at a given block. Whether an account
exists affects the gas cost of a transaction.


##### Example
```js
debug.accountExist("0x102e61f5d8f9bc71d0ad4a084df4e65e05ce0e1c", 1000)
> true
```
