package e2e

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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

type harness struct{}

func (h *harness) deployNativeContract(name string, code []byte) (*client.SendTransactionResponse, error) {
	return h.sendTransaction(builders.Transaction().
		WithMethod("_Deployments", "deployService").
		WithArgs(
			name,
			uint32(protocol.PROCESSOR_TYPE_NATIVE),
			code,
		).Builder())
}

func newHarness() *harness {
	return &harness{}
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

	// TODO - see issue https://github.com/orbs-network/orbs-network-go/issues/523
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted {
		return nil, errors.Errorf("got http status code %v calling %s", res.StatusCode, endpoint)
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
