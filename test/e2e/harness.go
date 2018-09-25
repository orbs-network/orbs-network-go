package e2e

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type E2EConfig struct {
	Bootstrap   bool
	ApiEndpoint string
}

const LOCAL_NETWORK_SIZE = 3

func getConfig() E2EConfig {
	Bootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	ApiEndpoint := "http://localhost:8082/api/v1/" // 8080 is leader, 8082 is node-3

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
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		firstRandomPort := 20000 + r.Intn(40000)

		federationNodes := make(map[string]config.FederationNode)
		gossipPeers := make(map[string]config.GossipPeer)
		for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
			publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
			federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
			gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(firstRandomPort+i), "127.0.0.1")
		}

		logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

		processorArtifactPath, dirToCleanup := getProcessorArtifactPath()
		os.RemoveAll(dirToCleanup)

		leaderKeyPair := keys.Ed25519KeyPairForTests(0)
		for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
			nodeKeyPair := keys.Ed25519KeyPairForTests(i)
			node := bootstrap.NewNode(
				fmt.Sprintf(":%d", 8080+i),
				nodeKeyPair.PublicKey(),
				nodeKeyPair.PrivateKey(),
				federationNodes,
				gossipPeers,
				uint16(firstRandomPort+i),
				leaderKeyPair.PublicKey(),
				consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
				logger,
				processorArtifactPath,
			)

			nodes = append(nodes, node)
		}
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
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}

func (h *harness) sendTransaction(t *testing.T, txBuilder *protocol.SignedTransactionBuilder) (*client.SendTransactionResponse, error) {
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: txBuilder,
	}).Build()
	responseBytes := h.httpPost(t, request, "send-transaction")
	response := client.SendTransactionResponseReader(responseBytes)
	if !response.IsValid() {
		// TODO: this is temporary until httpserver returns errors according to spec (issue #190)
		return nil, errors.Errorf("SendTransaction response invalid, raw as text: %s, raw as hex: %s, txHash: %s", string(responseBytes), hex.EncodeToString(responseBytes), digest.CalcTxHash(request.SignedTransaction().Transaction()))
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
	require.Equal(t, http.StatusOK, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	return bytes
}

func getProcessorArtifactPath() (string, string) {
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp")
	return filepath.Join(dir, "processor-artifacts"), dir
}
