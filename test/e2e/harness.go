package e2e

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/pkg/errors"
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

type harness struct {
	nodes []bootstrap.Node
}

func newHarness() *harness {
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

	return &harness{
		nodes: nodes,
	}
}

func (h *harness) gracefulShutdown() {
	if getConfig().Bootstrap {
		for _, node := range h.nodes {
			node.GracefulShutdown(1 * time.Second)
		}
	}
}

func (h *harness) sendTransaction(t *testing.T, txBuilder *protocol.SignedTransactionBuilder) (*client.SendTransactionResponse, error) {
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: txBuilder,
	}).Build()
	responseBytes := h.httpPost(t, request, "send-transaction")
	response := client.SendTransactionResponseReader(responseBytes)
	if !response.IsValid() {
		// TODO: this is temporary until httpserver returns errors according to spec (issue #190)
		return nil, errors.Errorf("SendTransaction response invalid, raw as text: %s, raw as hex: %s", string(responseBytes), hex.EncodeToString(responseBytes))
	}
	return response, nil
}

func (h *harness) callMethod(t *testing.T, txBuilder *protocol.TransactionBuilder) (*client.CallMethodResponse, error) {
	request := (&client.CallMethodRequestBuilder{
		Transaction: txBuilder,
	}).Build()
	responseBytes := h.httpPost(t, request, "call-method")
	response := client.CallMethodResponseReader(responseBytes)
	if !response.IsValid() {
		// TODO: this is temporary until httpserver returns errors according to spec (issue #190)
		return nil, errors.Errorf("CallMethod response invalid, raw as text: %s, raw as hex: %s", string(responseBytes), hex.EncodeToString(responseBytes))
	}
	return response, nil
}

func (h *harness) httpPost(t *testing.T, input membuffers.Message, method string) []byte {
	res, err := http.Post(getConfig().ApiEndpoint+method, "application/octet-stream", bytes.NewReader(input.Raw()))
	require.NoError(t, err)
	require.Equal(t, res.StatusCode, http.StatusOK)

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	return bytes
}
