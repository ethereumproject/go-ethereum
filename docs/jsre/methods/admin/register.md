#### admin.register

admin.register(address, contractaddress, contenthash);

will register content hash to the codehash (hash of the code of the contract on contractaddress). The register transaction is sent from the address in the first parameter. The transaction needs to be processed and confirmed on the canonical chain for the registration to take effect.

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
```
