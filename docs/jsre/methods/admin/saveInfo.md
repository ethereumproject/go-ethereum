#### admin.saveInfo

admin.saveInfo(contract.info, filename);

will write [contract info json](./Contracts-and-Transactions#contract-info-metadata) into the target file, calculates its content hash. This content hash then can used to associate a public url with where the contract info is publicly available and verifiable. If you register the codehash (hash of the code of the contract on contractaddress).

##### Returns

`contenthash` on success, otherwise `undefined`.

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
```
