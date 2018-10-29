# State Accumulation Over Time Tests

### TestSimulateStateInitFlowForSixMonthsAt100Tps

The purpose of this test is to anticipate the load of maintaining all state in memory 
without persisting anything to disk over a period of 6 months.

Two aspects may be impacted by running for significant amount of time without persistent state storage:
* Node startup time may be impacted since the state must be rebuilt from block storage
* Memory consumption may exceed the RAM requirement from a node

The test simulates the startup process of state accumulated over six months under these assumptiuons:
* 1 contract with a uint balance entry per user
* 1 Million users with 
* State modifications accumulated over 6 months at a transaction rate of 100TPS 
transactions per second.

The test distinguishes between the time spent processing transactions by the state storage service and 
time spent generating a random set of test data.

In reality another significant factor will be the latency and IO performance of reading
blocks from block storage to play against the state storage service. These overheads are outside the scope of this test
