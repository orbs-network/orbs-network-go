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
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type E2EConfig struct {
	Bootstrap   bool
	ApiEndpoint string
	baseUrl     string
}

const LOCAL_NETWORK_SIZE = 3
const START_HTTP_PORT = 8090

func getConfig() E2EConfig {
	Bootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	baseUrl := fmt.Sprintf("http://localhost:%d", START_HTTP_PORT+2) // 8080 is leader, 8082 is node-3
	ApiEndpoint := fmt.Sprintf("%s/api/v1/", baseUrl)

	if !Bootstrap {
		ApiEndpoint = os.Getenv("API_ENDPOINT")
	}

	return E2EConfig{
		Bootstrap,
		ApiEndpoint,
		baseUrl,
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

		os.MkdirAll(config.GetProjectSourceRootPath()+"/logs", 0755)

		logger := log.GetLogger().WithTags(
			log.String("_test", "e2e"),
			log.String("_branch", os.Getenv("GIT_BRANCH")),
			log.String("_commit", os.Getenv("GIT_COMMIT"))).
			WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

		processorArtifactPath, dirToCleanup := getProcessorArtifactPath()
		os.RemoveAll(dirToCleanup)

		leaderKeyPair := keys.Ed25519KeyPairForTests(0)
		for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
			nodeKeyPair := keys.Ed25519KeyPairForTests(i)

			logFile, err := os.OpenFile(fmt.Sprintf("%s/logs/node%d-%v.log", config.GetProjectSourceRootPath(), i+1, time.Now().Format(time.RFC3339Nano)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}

			nodeLogger := logger.WithOutput(log.NewOutput(logFile).WithFormatter(log.NewJsonFormatter()))

			cfg := config.ForProduction(processorArtifactPath)
			cfg.OverrideNodeSpecificValues(
				federationNodes,
				gossipPeers,
				uint16(firstRandomPort+i),
				nodeKeyPair.PublicKey(),
				nodeKeyPair.PrivateKey(),
				leaderKeyPair.PublicKey(),
				consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)

			node := bootstrap.NewNode(cfg, nodeLogger, fmt.Sprintf(":%d", START_HTTP_PORT+i))

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
			node.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
		}
	}
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}

func (h *harness) sendTransaction(txBuilder *protocol.SignedTransactionBuilder) (*client.SendTransactionResponse, error) {
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: txBuilder,
	}).Build()
	responseBytes, err := h.httpPost(request, "send-transaction")
	if err != nil {
		return nil, err
	}

	response := client.SendTransactionResponseReader(responseBytes)
	if !response.IsValid() {
		// TODO: this is temporary until httpserver returns errors according to spec (issue #190)
		return nil, errors.Errorf("SendTransaction response invalid, raw as text: %s, raw as hex: %s, txHash: %s", string(responseBytes), hex.EncodeToString(responseBytes), digest.CalcTxHash(request.SignedTransaction().Transaction()))
	}
	return response, nil
}

func (h *harness) callMethod(txBuilder *protocol.TransactionBuilder) (*client.CallMethodResponse, error) {
	request := (&client.CallMethodRequestBuilder{
		Transaction: txBuilder,
	}).Build()
	responseBytes, err := h.httpPost(request, "call-method")
	if err != nil {
		return nil, err
	}

	response := client.CallMethodResponseReader(responseBytes)
	if !response.IsValid() {
		// TODO: this is temporary until httpserver returns errors according to spec (issue #190)
		return nil, errors.Errorf("CallMethod response invalid, raw as text: %s, raw as hex: %s", string(responseBytes), hex.EncodeToString(responseBytes))
	}
	return response, nil
}

func (h *harness) httpPost(input membuffers.Message, endpoint string) ([]byte, error) {
	res, err := http.Post(h.apiUrlFor(endpoint), "application/octet-stream", bytes.NewReader(input.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got http status code %s calling %s", res.StatusCode, endpoint)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (h *harness) absoluteUrlFor(endpoint string) string {
	return getConfig().baseUrl + "/" + endpoint
}

func (h *harness) apiUrlFor(endpoint string) string {
	return getConfig().ApiEndpoint + endpoint
}

func getProcessorArtifactPath() (string, string) {
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp")
	return filepath.Join(dir, "processor-artifacts"), dir
}
