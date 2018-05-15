
#### admin.registerUrl

admin.registerUrl(address, codehash, contenthash);

this will register a contant hash to the contract' codehash. This will be used to locate [contract info json](./Contracts-and-Transactions#contract-info-metadata)
files. Address in the first parameter will be used to send the transaction.

##### Returns

`true` on success, otherwise `false`.

##### Examples

```js
source = "contract test {\n" +
"   /// @notice will multiply `a` by 7.\n" +
"   function multiply(uint a) returns(uint d) {\n" +
"      return a * 7;\n" +
"   }\n" +
"} ";
contract = eth.compile.solidity(source).test;
txhash = eth.sendTransaction({from: primary, data: contract.code });
// after it is uncluded
contractaddress = eth.getTransactionReceipt(txhash);
filename = "/tmp/info.json";
contenthash = admin.saveInfo(contract.info, filename);
admin.register(primary, contractaddress, contenthash);
admin.registerUrl(primary, contenthash, "file://"+filename);
```
