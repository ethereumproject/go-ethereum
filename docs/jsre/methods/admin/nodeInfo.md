
#### admin.nodeInfo

admin.nodeInfo

##### Returns

information on the node.

##### Example

```
> admin.nodeInfo
{
Name: 'Ethereum(G)/v0.9.36/darwin/go1.4.1',
NodeUrl: 'enode://c32e13952965e5f7ebc85b02a2eb54b09d55f553161c6729695ea34482af933d0a4b035efb5600fc5c3ea9306724a8cbd83845bb8caaabe0b599fc444e36db7e@89.42.0.12:30303',
NodeID: '0xc32e13952965e5f7ebc85b02a2eb54b09d55f553161c6729695ea34482af933d0a4b035efb5600fc5c3ea9306724a8cbd83845bb8caaabe0b599fc444e36db7e',
IP: '89.42.0.12',
DiscPort: 30303,
TCPPort: 30303,
Td: '0',
ListenAddr: '[::]:30303'
}
```

To connect to a node, use the [enode-format](https://github.com/ethereumproject/wiki/wiki/enode-url-format) nodeUrl as an argument to [addPeer](#adminaddpeer) or with CLI param `bootnodes`.
