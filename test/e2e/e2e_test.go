package e2e

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

type E2EConfig struct {
	Bootstrap   bool
	ApiEndpoint string
}

func getConfig() E2EConfig {
	Bootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	ApiEndpoint := "http://localhost:8080/api/"

	if !Bootstrap {
		ApiEndpoint = os.Getenv("API_ENDPOINT")
	}

	return E2EConfig{
		Bootstrap,
		ApiEndpoint,
	}
}

func TestOrbsNetworkAcceptsTransactionAndCommitsIt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	var nodes []bootstrap.Node

	// TODO: kill me - why do we need this override?
	if getConfig().Bootstrap {
		gossipTransport := gossipAdapter.NewTamperingTransport()

		federationNodes := make(map[string]config.FederationNode)
		leaderKeyPair := keys.Ed25519KeyPairForTests(0)
		for i := 0; i < 3; i++ {
			nodeKeyPair := keys.Ed25519KeyPairForTests(i)
			federationNodes[nodeKeyPair.PublicKey().KeyForMap()] = config.NewHardCodedFederationNode(nodeKeyPair.PublicKey())
		}

		logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

		for i := 0; i < 3; i++ {
			nodeKeyPair := keys.Ed25519KeyPairForTests(i)
			node := bootstrap.NewNode(
				fmt.Sprintf(":%d", 8080+i),
				nodeKeyPair.PublicKey(),
				nodeKeyPair.PrivateKey(),
				federationNodes,
				leaderKeyPair.PublicKey(),
				consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
				logger,
				gossipTransport,
			)

			nodes = append(nodes, node)
		}

		// To let node start up properly, otherwise in Docker we get connection refused
		time.Sleep(100 * time.Millisecond)
	}

	amount := uint64(99)
	tx := builders.TransferTransaction().WithAmount(amount).Builder()

	_ = sendTransaction(t, tx)

	m := &protocol.TransactionBuilder{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	ok := test.Eventually(test.EVENTUALLY_DOCKER_E2E_TIMEOUT, func() bool {
		outputArgsIterator := builders.ClientCallMethodResponseOutputArgumentsParse(callMethod(t, m).ClientResponse)
		if outputArgsIterator.HasNext() {
			return outputArgsIterator.NextArguments().Uint64Value() == amount
		} else {
			return false
		}
	})

	require.True(t, ok, "should return expected amount from BenchmarkToken.getBalance()")

	if getConfig().Bootstrap {
		for _, node := range nodes {
			node.GracefulShutdown(1 * time.Second)
		}
	}
}

func sendTransaction(t *testing.T, txBuilder *protocol.SignedTransactionBuilder) *services.SendTransactionOutput {
	input := (&client.SendTransactionRequestBuilder{
		SignedTransaction: txBuilder,
	}).Build()

	return &services.SendTransactionOutput{ClientResponse: client.SendTransactionResponseReader(httpPost(t, input, "send-transaction"))}
}

func callMethod(t *testing.T, txBuilder *protocol.TransactionBuilder) *services.CallMethodOutput {
	input := (&client.CallMethodRequestBuilder{
		Transaction: txBuilder,
	}).Build()

	return &services.CallMethodOutput{ClientResponse: client.CallMethodResponseReader(httpPost(t, input, "call-method"))}

}

func httpPost(t *testing.T, input membuffers.Message, method string) []byte {
	res, err := http.Post(getConfig().ApiEndpoint+method, "application/octet-stream", bytes.NewReader(input.Raw()))
	require.NoError(t, err)
	require.Equal(t, res.StatusCode, http.StatusOK)

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	return bytes
}
