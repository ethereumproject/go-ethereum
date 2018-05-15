
#### miner.startAutoDAG

miner.startAutoDAG()

Starts automatic pregeneration of the [ethash DAG](https://github.com/ethereumproject/wiki/wiki/Ethash-DAG). This process make sure that the DAG for the subsequent epoch is available allowing mining right after the new epoch starts. If this is used by most network nodes, then blocktimes are expected to be normal at epoch transition. Auto DAG is switched on automatically when mining is started and switched off when the miner stops.

##### Returns

`true` on success, otherwise `false`.
