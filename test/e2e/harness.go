// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type E2EConfig struct {
	AppVcid primitives.VirtualChainId
	RemoteEnvironment bool
	Bootstrap         bool
	AppChainUrl       string
	StressTest        StressTestConfig
	EthereumEndpoint  string
	IsExperimental    bool
}

type StressTestConfig struct {
	enabled               bool
	numberOfTransactions  int64
	acceptableFailureRate int64
	targetTPS             float64
}

const START_HTTP_PORT = 8090
const START_GOSSIP_PORT = 8190

type Harness struct {
	client     *orbsClient.OrbsClient
	metricsUrl string
	config     *E2EConfig
}

func NewAppHarness() *Harness {
	config := GetConfig()

	return &Harness{
		client:     orbsClient.NewClient(config.AppChainUrl, uint32(config.AppVcid), codec.NETWORK_TYPE_TEST_NET),
		metricsUrl: config.AppChainUrl + "/metrics",
		config:     &config,
	}
}

func isNotHttp202Error(err error) bool {
	return !strings.Contains(err.Error(), "http status 202 Accepted")
}

func (h *Harness) DeployContract(from *keys.Ed25519KeyPair, contractName string, processorType orbsClient.ProcessorType, code ...[]byte) (*codec.TransactionResponse, error) {
	timeoutDuration := 15 * time.Second
	beginTime := time.Now()

	txOut, txId, err := h.SendDeployTransaction(from.PublicKey(), from.PrivateKey(), contractName, processorType, code...)
	if err != nil && isNotHttp202Error(err) { // TODO the SDK treats HTTP 202 as an error
		return nil, errors.Wrap(err, "failed to deploy native contract")
	}

	for txOut.TransactionStatus == codec.TRANSACTION_STATUS_PENDING {
		// check timeout
		if time.Now().Sub(beginTime) > timeoutDuration {
			return nil, fmt.Errorf("contract deployment is TRANSACTION_STATUS_PENDING for over %v", timeoutDuration)
		}

		time.Sleep(10 * time.Millisecond)

		txOut, _ = h.GetTransactionStatus(txId)
	}

	return txOut, err
}

func (h *Harness) DeployNativeContract(from *keys.Ed25519KeyPair, contractName string, code ...[]byte) (*codec.TransactionResponse, error) {
	return h.DeployContract(from, contractName, orbsClient.PROCESSOR_TYPE_NATIVE, code...)
}

func (h *Harness) SendTransaction(senderPublicKey []byte, senderPrivateKey []byte, contractName string, methodName string, args ...interface{}) (*codec.TransactionResponse, string, error) {
	payload, txId, err := h.client.CreateTransaction(senderPublicKey, senderPrivateKey, contractName, methodName, args...)
	if err != nil {
		return nil, txId, err
	}
	out, err := h.client.SendTransaction(payload)
	return out.TransactionResponse, txId, err
}

func (h *Harness) SendDeployTransaction(senderPublicKey []byte, senderPrivateKey []byte, contractName string, processorType orbsClient.ProcessorType, code ...[]byte) (*codec.TransactionResponse, string, error) {
	payload, txId, err := h.client.CreateDeployTransaction(senderPublicKey, senderPrivateKey, contractName, processorType, code...)
	if err != nil {
		return nil, txId, err
	}
	out, err := h.client.SendTransaction(payload)
	return out.TransactionResponse, txId, err
}

func (h *Harness) runQueryAtBlockHeight(timeout time.Duration, expectedBlockHeight uint64, senderPublicKey []byte, contractName string, methodName string, args ...interface{}) (*codec.RunQueryResponse, error) {
	var lastErr error
	var response *codec.RunQueryResponse

	if test.Eventually(timeout, func() bool {
		var err error
		response, err = h.RunQuery(senderPublicKey, contractName, methodName, args...)
		if err != nil {
			lastErr = err
			return false
		}
		return response.BlockHeight >= expectedBlockHeight // An error could be a result of the contract not being deployed at the currently synced block height, suppress it unless not eventually successful
	}) {
		return response, nil
	}

	// Couldn't reach the block height

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, errors.Errorf("did not reach height %d before timeout (got last response at height %d)", expectedBlockHeight, response.BlockHeight)
}

func (h *Harness) RunQuery(senderPublicKey []byte, contractName string, methodName string, args ...interface{}) (response *codec.RunQueryResponse, err error) {
	payload, err := h.client.CreateQuery(senderPublicKey, contractName, methodName, args...)
	if err != nil {
		return nil, err
	}
	response, err = h.client.SendQuery(payload)
	return
}

func (h *Harness) GetTransactionStatus(txId string) (*codec.TransactionResponse, error) {
	response, err := h.client.GetTransactionStatus(txId)
	return response.TransactionResponse, err
}

func (h *Harness) GetTransactionReceiptProof(txId string) (response *codec.GetTransactionReceiptProofResponse, err error) {
	response, err = h.client.GetTransactionReceiptProof(txId)
	return
}

type metrics map[string]map[string]interface{}

func (h *Harness) GetMetrics() metrics {
	res, err := http.Get(h.metricsUrl)

	if err != nil {
		fmt.Println(h.metricsUrl, err)
	}

	if res == nil {
		return nil
	}

	readBytes, _ := ioutil.ReadAll(res.Body)
	m := make(metrics)
	_ = json.Unmarshal(readBytes, &m)

	return m
}

func (h *Harness) DeployContractAndRequireSuccess(t *testing.T, keyPair *keys.Ed25519KeyPair, contractName string, contractBytes ...[]byte) uint64 {

	h.WaitUntilTransactionPoolIsReady(t)

	result, dcErr := h.DeployNativeContract(keyPair, contractName, contractBytes...)

	require.Nil(t, dcErr, "expected deploy contract to succeed")
	require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED, result.TransactionStatus, "expected deploy contract to commit")
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS, result.ExecutionResult, "expected deploy contract to succeed")

	return result.BlockHeight
}

func (h *Harness) WaitUntilTransactionPoolIsReady(t *testing.T) {

	recentBlockTimeDiff := getE2ETransactionPoolNodeSyncRejectTime() / 2
	require.True(t, test.Eventually(15*time.Second, func() bool {

		m := h.GetMetrics()
		if m == nil {
			return false
		}

		lastCommittedTimestamp := int64(m["TransactionPool.LastCommitted.TimeNano"]["Value"].(float64))
		diff := lastCommittedTimestamp - time.Now().Add(recentBlockTimeDiff*-1).UnixNano()
		return diff >= 0
	}), "timed out waiting for a transaction pool to sync a recent block and begin accepting new tx")
}

func getE2ETransactionPoolNodeSyncRejectTime() time.Duration {
	return config.ForE2E(
		"",
		0,
		0,
		primitives.NodeAddress{},
		primitives.EcdsaSecp256K1PrivateKey{},
		nil,
		nil,
		"",
		"",
		"",
		primitives.NodeAddress{},
		0,
		"",
	).TransactionPoolNodeSyncRejectTime()
}

func PrintTestTime(t *testing.T, msg string, last *time.Time) {
	t.Logf("%s (+%.3fs)", msg, time.Since(*last).Seconds())
	*last = time.Now()
}

func (h *Harness) envSupportsTestingFileAssets() bool {
	return h.config.RemoteEnvironment
}

func GetConfig() E2EConfig {
	appVcid := primitives.VirtualChainId(42)

	circleTag := os.Getenv("CIRCLE_TAG")
	isExperimental := !strings.HasPrefix(circleTag, "v")

	if vcId, err := strconv.ParseUint(os.Getenv("VCHAIN"), 10, 0); err == nil {
		appVcid = primitives.VirtualChainId(vcId)
	}

	shouldBootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	appChainUrl := fmt.Sprintf("http://localhost:%d", START_HTTP_PORT+2) // 8090 is leader, 8082 is node-3

	isRemoteEnvironment := os.Getenv("REMOTE_ENV") == "true"

	stressTestEnabled := os.Getenv("STRESS_TEST") == "true"
	stressTestNumberOfTransactions := int64(10000)
	stressTestFailureRate := int64(2)
	stressTestTargetTPS := float64(700)

	ethereumEndpoint := "http://127.0.0.1:8545"

	if !shouldBootstrap {
		apiEndpoint := os.Getenv("API_ENDPOINT")
		appChainUrl = strings.TrimSuffix(strings.TrimRight(apiEndpoint, "/"), "/api/v1")
		ethereumEndpoint = os.Getenv("ETHEREUM_ENDPOINT")
	}

	if stressTestEnabled {
		stressTestNumberOfTransactions, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_NUMBER_OF_TRANSACTIONS"), 10, 0)
		stressTestFailureRate, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_FAILURE_RATE"), 10, 0)
		stressTestTargetTPS, _ = strconv.ParseFloat(os.Getenv("STRESS_TEST_TARGET_TPS"), 0)
	}

	return E2EConfig{
		AppVcid: appVcid,
		Bootstrap:         shouldBootstrap,
		RemoteEnvironment: isRemoteEnvironment,
		IsExperimental:    isExperimental,
		AppChainUrl:       appChainUrl,
		StressTest: StressTestConfig{
			enabled:               stressTestEnabled,
			numberOfTransactions:  stressTestNumberOfTransactions,
			acceptableFailureRate: stressTestFailureRate,
			targetTPS:             stressTestTargetTPS,
		},
		EthereumEndpoint: ethereumEndpoint,
	}
}

func requireSuccessful(t testing.TB, response *codec.TransactionResponse) {
	require.Equal(t, codec.TRANSACTION_STATUS_COMMITTED, response.TransactionStatus)
	require.Equal(t, codec.EXECUTION_RESULT_SUCCESS, response.ExecutionResult)
}
