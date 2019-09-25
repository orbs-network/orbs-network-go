package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestDeployContractAgainstDockerizedGammaInstance(t *testing.T) {
	rnd := rand.NewControlledRand(t)
	endpoint := os.Getenv("API_ENDPOINT")
	if endpoint == "" {
		t.Skip("API_ENDPOINT not provided, skipping docker-based test")
	}

	owner := keys.Ed25519KeyPairForTests(5)
	client := orbs.NewClient(endpoint, 42, codec.NETWORK_TYPE_TEST_NET)
	start := uint64(rnd.Int31())
	contractName := fmt.Sprintf("counter%d", start)

	tx, txId, err := client.CreateDeployTransaction(owner.PublicKey(), owner.PrivateKey(), contractName, orbs.PROCESSOR_TYPE_NATIVE, []byte(contracts.NativeSourceCodeForCounter(start)))
	require.NoError(t, err, "failed creating deploy transaction")

	require.True(t, test.Eventually(5*time.Second, func() bool {
		deployResponse, err := client.SendTransactionAsync(tx)
		json, _ := deployResponse.MarshalJSON()
		t.Log(string(json))
		if err != nil {
			return false
		}
		return deployResponse.RequestStatus == codec.REQUEST_STATUS_IN_PROCESS
	}))

	require.True(t, test.Eventually(5*time.Second, func() bool {
		statusResponse, err := client.GetTransactionStatus(txId)
		if err != nil {
			return false
		}

		json, _ := statusResponse.MarshalJSON()
		t.Log(string(json))
		return statusResponse.ExecutionResult == codec.EXECUTION_RESULT_SUCCESS
	}))

}
