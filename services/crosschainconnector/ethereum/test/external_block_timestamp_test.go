package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
	"time"
)

func moveToRealtimeInGanache(t *testing.T, h *harness, ctx context.Context, buffer int) {
	latestBlockInGanache, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "failed to get latest block in ganache")

	now := time.Now().Unix()
	gap := now - latestBlockInGanache.Time.Int64() - int64(buffer)
	require.True(t, gap >= 0, "ganache must be set up back enough to the past so finality test would pass in this flow, it was rolled too close to realtime, gap was %d", gap)
	t.Logf("moving %d blocks into the future to get to now - %d seconds", gap, buffer)
	h.moveBlocksInGanache(t, int(gap), 1)
}

//TODO this test does not deal well with gaps in ganache, meaning that a ganache that has been standing idle for more than a few seconds will fail this test while
// trying to assert that a contract cannot be called before it has been deployed
func TestFullFlowWithVaryingTimestamps(t *testing.T) {
	// the idea of this test is to make sure that the entire 'call-from-ethereum' logic works on a specific timestamp and different states in time (blocks)
	// it requires ganache or some other RPC backend to transact

	// 1. it **assumes** that ganache started in the past, and is not 'realtime' - it does not care how long but in order
	//    for it to work it must be greater than the finality value
	// 2. next it will write a block (to support working on a clean ganache)
	// 3. it will then find out how much it needs to fast forward to get to realtime, and will write a block per second to fill the gap (using evm_increaseTime, evm_mine)
	// 4. it does that to realtime - finality - some gap so we deploy our contract enough time in the past
	// 5. it will then deploy the contract
	// 6. it will then run to actual realtime
	// 6. it will attempt to call the contract at realtime, and then call it at the time it did not exists expecting an error

	if !runningWithDocker() {
		t.Skip("this test relies on external components - ganache, and will be skipped unless running in docker")
	}

	test.WithContext(func(ctx context.Context) {
		h := newRpcEthereumConnectorHarness(t, ConfigForExternalRPCConnection())
		finalityComponent := uint32(h.config.finalityTimeComponent.Seconds()) + h.config.finalityBlocksComponent

		// create first block in case we are running only this test (clean ganache, but no real hard in actual)
		h.moveBlocksInGanache(t, 1, 1)
		latestBlockInGanache, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "failed to get latest block in ganache")
		t.Logf("starting point in ganache is %d | %d", latestBlockInGanache.Number.Int64(), latestBlockInGanache.Time.Int64())

		// time now, gap to ganache time, catch up
		buffer := finalityComponent + 10 // +10 gap as described above
		moveToRealtimeInGanache(t, h, ctx, int(buffer))

		latestBlockInGanache, err = h.rpcAdapter.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "failed to get latest block in ganache")
		t.Logf("time and block -%d secs %d | %d", finalityComponent+10, latestBlockInGanache.Number.Int64(), latestBlockInGanache.Time.Int64())
		timeBeforeContractWasDeployed := time.Unix(latestBlockInGanache.Time.Int64(), 0)

		// deploy takes a while, over a sec, so there is no need to sleep here, the next block will be created okay,
		// it might take even two secs, which will break our 'relatime' catchup,
		// but its okay to have ganache one or two secs into the future (finality will cover for us)
		expectedTextFromEthereum := "test3"
		// ========== this will advance one block as well for the delpoy
		contractAddress, err := h.deployRpcStorageContract(expectedTextFromEthereum)
		require.NoError(t, err, "failed deploying contract to Ethereum")

		latestBlockInGanacheAfterDeploy, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "failed to get latest block in ganache after deploy")
		t.Logf("block in ganache when deployed %d | %d", latestBlockInGanacheAfterDeploy.Number.Int64(), latestBlockInGanacheAfterDeploy.Time.Int64())

		// pushing to 'realtime'
		moveToRealtimeInGanache(t, h, ctx, 0)

		actualBlockInGanache, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
		require.NoError(t, err, "failed to get latest block in ganache")
		t.Logf("time and block in actual %d | %d, system time %d", actualBlockInGanache.Number.Int64(), actualBlockInGanache.Time.Int64(), time.Now().Unix())

		// =============== test starts here

		methodToCall := "getValues"
		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "abi parse failed for simple storage contract")

		ethCallData, err := ethereum.ABIPackFunctionInputArguments(parsedABI, methodToCall, nil)
		require.NoError(t, err, "this means we couldn't pack the params for ethereum, something is broken with the harness")

		// request at time now, which should be (with finality) after the contract was deployed
		input := builders.EthereumCallContractInput().
			WithContractAddress(contractAddress).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err := h.connector.EthereumCallContract(ctx, input)
		require.NoError(t, err, "expecting call to succeed")
		require.True(t, len(output.EthereumAbiPackedOutput) > 0, "expecting output to have some data")
		ret := new(struct { // this is the expected return type of that ethereum call for the SimpleStorage contract getValues
			IntValue    *big.Int
			StringValue string
		})

		ethereum.ABIUnpackFunctionOutputArguments(parsedABI, ret, methodToCall, output.EthereumAbiPackedOutput)
		require.Equal(t, expectedTextFromEthereum, ret.StringValue, "text part from eth")

		input = builders.EthereumCallContractInput().
			WithTimestamp(timeBeforeContractWasDeployed).
			WithContractAddress(contractAddress).
			WithAbi(contract.SimpleStorageABI).
			WithFunctionName(methodToCall).
			WithPackedArguments(ethCallData).
			Build()

		output, err = h.connector.EthereumCallContract(ctx, input)
		require.Error(t, err, "expecting call to fail as contract is not yet deployed in a past time block")
	})
}
