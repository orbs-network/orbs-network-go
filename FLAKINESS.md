# Flakiness

Orbs is a highly distributed system and as such prone to synchronization challenges, race conditions and such. This means some issues are not exposed in tests deterministically.

## Reporting flakiness

When encoutering a new case of suspected flakiness, please follow these steps:

1. Make sure your branch is updated to the latest master.

2. Search [flakiness labeled issues](https://github.com/orbs-network/orbs-network-go/labels/flakiness) for the **test name** which you find flaky. If you find it, add a comment there instead of opening a new issue.

3. Open a new issue, but don't forget any of these:

    * Add the label **flakiness**
    
    * Make sure the issue name contains the **test name**
        > for example: `TestInterNodeBlockSync_WithBenchmarkConsensusBlocks is flaky because sync is stuck`
    
    * Make sure the issue body contains a link to the failing **CircleCI build**
        > for example: https://circleci.com/gh/orbs-network/orbs-network-go/8522
        
    * Copy the few lines of failure from the logs and paste it in the issue body
        > for example:
        ```
        --- FAIL: TestContract_SendBroadcast/DirectTransport (1.14s)
                require.go:794: 
                      Error Trace:	transport_contract_test.go:131
                                          transport_contract_test.go:52
                                          context.go:11
                                          transport_contract_test.go:37
                      Error:      	Received unexpected error:
                                    Function #1 OnTransportMessageReceived executed 0 times, expected: 1
                      Test:       	TestContract_SendBroadcast/DirectTransport
        ```
