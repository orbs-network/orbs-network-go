package e2e

import (
	"bytes"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_IPFSTemp"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIPFSSystemContract(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := NewAppHarness()
		lt := time.Now()
		PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		PrintTestTime(t, "first block committed", &lt)

		contractName := ipfs_systemcontract.CONTRACT_NAME

		PrintTestTime(t, "send deploy - start", &lt)

		PrintTestTime(t, "send deploy - end", &lt)

		// check counter
		ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
			PrintTestTime(t, "run query - start", &lt)
			response, err2 := h.RunQuery(OwnerOfAllSupply.PublicKey(), contractName, "read", []byte("any-hash"))
			PrintTestTime(t, "run query - end", &lt)

			if err2 == nil && response.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS {
				return bytes.Equal(response.OutputArguments[0].([]byte), []byte("Diamond Dogs"))
			}
			return false
		})
		require.True(t, ok, "get counter should return counter start")
	})
}
