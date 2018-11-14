package e2e

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type E2EConfig struct {
	bootstrap   bool
	apiEndpoint string
	baseUrl     string

	stressTest StressTestConfig
}

type StressTestConfig struct {
	enabled               bool
	numberOfTransactions  int64
	acceptableFailureRate int64
	targetTPS             float64
}

const LOCAL_NETWORK_SIZE = 3
const START_HTTP_PORT = 8090

func getConfig() E2EConfig {
	shouldBootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	baseUrl := fmt.Sprintf("http://localhost:%d", START_HTTP_PORT+2) // 8080 is leader, 8082 is node-3
	apiEndpoint := fmt.Sprintf("%s/api/v1/", baseUrl)

	stressTestEnabled := os.Getenv("STRESS_TEST") == "true"
	stressTestNumberOfTransactions := int64(10000)
	stressTestFailureRate := int64(2)
	stressTestTargetTPS := float64(700)

	if !shouldBootstrap {
		apiEndpoint = os.Getenv("API_ENDPOINT")
		apiUrl, _ := url.Parse(apiEndpoint)
		baseUrl = apiUrl.Scheme + "://" + apiUrl.Host

		if stressTestEnabled {
			stressTestNumberOfTransactions, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_NUMBER_OF_TRANSACTIONS"), 10, 0)
			stressTestFailureRate, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_FAILURE_RATE"), 10, 0)
			stressTestTargetTPS, _ = strconv.ParseFloat(os.Getenv("STRESS_TEST_TARGET_TPS"), 0)
		}
	}

	return E2EConfig{
		shouldBootstrap,
		apiEndpoint,
		baseUrl,
		StressTestConfig{
			stressTestEnabled,
			stressTestNumberOfTransactions,
			stressTestFailureRate,
			stressTestTargetTPS,
		},
	}
}

type inProcessE2ENetwork struct {
	nodes []bootstrap.Node
}

func (h *inProcessE2ENetwork) gracefulShutdown() {
	if getConfig().bootstrap {
		for _, node := range h.nodes {
			node.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
		}
	}
}

func newInProcessE2ENetwork() *inProcessE2ENetwork {
	return &inProcessE2ENetwork{bootstrapNetwork()}
}

type harness struct {}

func newHarness() *harness {
	return &harness{}
}

func bootstrapNetwork() (nodes []bootstrap.Node) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstRandomPort := 20000 + r.Intn(40000)
	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(firstRandomPort+i), "127.0.0.1")
	}
	os.MkdirAll(config.GetProjectSourceRootPath()+"/_logs", 0755)
	logger := log.GetLogger().WithTags(
		log.String("_test", "e2e"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT"))).
		WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	leaderKeyPair := keys.Ed25519KeyPairForTests(0)
	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)

		logFile, err := os.OpenFile(fmt.Sprintf("%s/_logs/node%d-%v.log", config.GetProjectSourceRootPath(), i+1, time.Now().Format(time.RFC3339Nano)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}

		nodeLogger := logger.WithOutput(log.NewFormattingOutput(logFile, log.NewJsonFormatter()))
		processorArtifactPath, _ := getProcessorArtifactPath()

		cfg := config.ForE2E(processorArtifactPath)
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
	return nodes
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
	res, err := http.Post(h.apiUrlFor(endpoint), "application/membuffers", bytes.NewReader(input.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got http status code %s calling %s", res.StatusCode, endpoint)
	}

	readBytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return readBytes, nil
}

func (h *harness) absoluteUrlFor(endpoint string) string {
	return getConfig().baseUrl + "/" + endpoint
}

func (h *harness) apiUrlFor(endpoint string) string {
	return getConfig().apiEndpoint + endpoint
}

func getProcessorArtifactPath() (string, string) {
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp")
	return filepath.Join(dir, "processor-artifacts"), dir
}

type metrics map[string]map[string]interface{}

func (h *harness) getMetrics() metrics {
	res, err := http.Get(h.absoluteUrlFor("/metrics"))

	if err != nil {
		fmt.Println(h.absoluteUrlFor("/metrics"), err)
	}

	if res == nil {
		return nil
	}

	readBytes, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(readBytes))

	m := make(metrics)
	json.Unmarshal(readBytes, &m)

	return m
}

func printTestTime(t *testing.T, msg string, last *time.Time) {
	t.Logf("%s (+%.3fs)", msg, time.Since(*last).Seconds())
	*last = time.Now()
}
