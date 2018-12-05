package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

type E2EConfig struct {
	bootstrap  bool
	baseUrl    string
	stressTest StressTestConfig
}

type StressTestConfig struct {
	enabled               bool
	numberOfTransactions  int64
	acceptableFailureRate int64
	targetTPS             float64
}

const VITRUAL_CHAIN_ID = 42
const START_HTTP_PORT = 8090

type harness struct {
	client *orbsclient.OrbsClient
}

func newHarness() *harness {
	return &harness{
		client: orbsclient.NewOrbsClient(getConfig().baseUrl, VITRUAL_CHAIN_ID, codec.NETWORK_TYPE_TEST_NET),
	}
}

func (h *harness) deployNativeContract(from *keys.Ed25519KeyPair, contractName string, code []byte) (*codec.SendTransactionResponse, error) {
	response, _, err := h.sendTransaction(from, "_Deployments", "deployService", contractName, uint32(protocol.PROCESSOR_TYPE_NATIVE), code)
	return response, err
}

func (h *harness) sendTransaction(sender *keys.Ed25519KeyPair, contractName string, methodName string, args ...interface{}) (response *codec.SendTransactionResponse, txId string, err error) {
	payload, txId, err := h.client.CreateSendTransactionPayload(sender.PublicKey(), sender.PrivateKey(), contractName, methodName, args...)
	if err != nil {
		return nil, txId, err
	}
	response, err = h.client.SendTransaction(payload)
	return
}

func (h *harness) callMethod(sender *keys.Ed25519KeyPair, contractName string, methodName string, args ...interface{}) (response *codec.CallMethodResponse, err error) {
	payload, err := h.client.CreateCallMethodPayload(sender.PublicKey(), contractName, methodName, args...)
	if err != nil {
		return nil, err
	}
	response, err = h.client.CallMethod(payload)
	return
}

func (h *harness) getTransactionStatus(txId string) (response *codec.GetTransactionStatusResponse, err error) {
	payload, err := h.client.CreateGetTransactionStatusPayload(txId)
	if err != nil {
		return nil, err
	}
	response, err = h.client.GetTransactionStatus(payload)
	return
}

func (h *harness) absoluteUrlFor(endpoint string) string {
	return getConfig().baseUrl + "/" + endpoint
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

func (h *harness) waitUntilTransactionPoolIsReady(t *testing.T) {
	require.True(t, test.Eventually(3*time.Second, func() bool { // 3 seconds to avoid jitter but it really shouldn't take that long
		m := h.getMetrics()
		if m == nil {
			return false
		}

		blockHeight := m["TransactionPool.BlockHeight"]["Value"].(float64)

		return blockHeight > 0
	}), "could not retrieve metrics")
}

func printTestTime(t *testing.T, msg string, last *time.Time) {
	t.Logf("%s (+%.3fs)", msg, time.Since(*last).Seconds())
	*last = time.Now()
}

func getConfig() E2EConfig {
	shouldBootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	baseUrl := fmt.Sprintf("http://localhost:%d", START_HTTP_PORT+2) // 8080 is leader, 8082 is node-3

	stressTestEnabled := os.Getenv("STRESS_TEST") == "true"
	stressTestNumberOfTransactions := int64(10000)
	stressTestFailureRate := int64(2)
	stressTestTargetTPS := float64(700)

	if !shouldBootstrap {
		apiEndpoint := os.Getenv("API_ENDPOINT")
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
		baseUrl,
		StressTestConfig{
			stressTestEnabled,
			stressTestNumberOfTransactions,
			stressTestFailureRate,
			stressTestTargetTPS,
		},
	}
}
