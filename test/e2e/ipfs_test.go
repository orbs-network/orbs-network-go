package e2e

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-client-sdk-go/orbs"
	ipfsTest "github.com/orbs-network/orbs-network-go/services/ipfs/test"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"
	"time"
)

func TestIPFSProxyContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {
		ipfsDaemonHarness := ipfsTest.NewIPFSDaemonHarness()
		require.NoError(t, ipfsDaemonHarness.StartDaemon())
		defer ipfsDaemonHarness.StopDaemon()

		require.NoError(t, ipfsDaemonHarness.AddFile(ipfsTest.ExampleJSONPath()))

		h := NewAppHarness()
		lt := time.Now()
		PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		PrintTestTime(t, "first block committed", &lt)

		contractName := fmt.Sprintf("IPFSProxy%d", time.Now().UnixNano())

		PrintTestTime(t, "send deploy - start", &lt)

		sources, err := orbs.ReadSourcesFromDir("./contracts/ipfs_proxy")
		require.NoError(t, err)

		h.DeployContractAndRequireSuccess(t, OwnerOfAllSupply, contractName, sources...)

		PrintTestTime(t, "send deploy - start", &lt)

		PrintTestTime(t, "send deploy - end", &lt)

		contents, _ := ioutil.ReadFile(ipfsTest.ExampleJSONPath())

		// check contents
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(OwnerOfAllSupply.PublicKey(), contractName, "read", ipfsTest.EXAMPLE_JSON_HASH)
			PrintTestTime(t, "run query - end", &lt)

			resJSON, _ := response.MarshalJSON()
			println("response_", string(resJSON))
			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return bytes.Equal(response.OutputArguments[0].([]byte), contents)
			}
			return false
		})
		require.True(t, ok)
	})
}
