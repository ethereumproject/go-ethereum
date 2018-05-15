
#### admin.getContractInfo

admin.getContractInfo(address)

this will retrieve the [contract info json](./Contracts-and-Transactions#contract-info-metadata) for a contract on the address

##### Returns

returns the contract info object

##### Examples

```js
> info = admin.getContractInfo(contractaddress)
> source = info.source
> abi = info.abiDefinition
```
