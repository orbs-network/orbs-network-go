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
	virtualChainId    uint32
	remoteEnvironment bool
	bootstrap         bool
	baseUrl           string
	stressTest        StressTestConfig
	ethereumEndpoint  string
}

type StressTestConfig struct {
	enabled               bool
	numberOfTransactions  int64
	acceptableFailureRate int64
	targetTPS             float64
}

const START_HTTP_PORT = 8090

type harness struct {
	client *orbsClient.OrbsClient
	config *E2EConfig
}

func newHarness() *harness {
	config := getConfig()

	return &harness{
		client: orbsClient.NewClient(config.baseUrl, config.virtualChainId, codec.NETWORK_TYPE_TEST_NET),
		config: &config,
	}
}

func (h *harness) deployNativeContract(from *keys.Ed25519KeyPair, contractName string, code ...[]byte) (codec.ExecutionResult, codec.TransactionStatus, error) {
	timeoutDuration := 10 * time.Second
	beginTime := time.Now()

	sendTxOut, txId, err := h.sendDeployTransaction(from.PublicKey(), from.PrivateKey(), contractName, code...)

	if err != nil {
		return "", "", errors.Wrap(err, "failed to deploy native contract")
	}

	txStatus, executionResult := sendTxOut.TransactionStatus, sendTxOut.ExecutionResult

	for txStatus == codec.TRANSACTION_STATUS_PENDING {
		// check timeout
		if time.Now().Sub(beginTime) > timeoutDuration {
			return "", "", fmt.Errorf("contract deployment is TRANSACTION_STATUS_PENDING for over %v", timeoutDuration)
		}

		time.Sleep(10 * time.Millisecond)

		txStatusOut, _ := h.getTransactionStatus(txId)

		txStatus, executionResult = txStatusOut.TransactionStatus, txStatusOut.ExecutionResult
	}

	return executionResult, txStatus, err
}

func (h *harness) sendTransaction(senderPublicKey []byte, senderPrivateKey []byte, contractName string, methodName string, args ...interface{}) (response *codec.SendTransactionResponse, txId string, err error) {
	payload, txId, err := h.client.CreateTransaction(senderPublicKey, senderPrivateKey, contractName, methodName, args...)
	if err != nil {
		return nil, txId, err
	}
	response, err = h.client.SendTransaction(payload)
	return
}

func (h *harness) sendDeployTransaction(senderPublicKey []byte, senderPrivateKey []byte, contractName string, code ...[]byte) (response *codec.SendTransactionResponse, txId string, err error) {
	payload, txId, err := h.client.CreateDeployTransaction(senderPublicKey, senderPrivateKey, contractName, orbsClient.PROCESSOR_TYPE_NATIVE, code...)
	if err != nil {
		return nil, txId, err
	}
	response, err = h.client.SendTransaction(payload)
	return
}

func (h *harness) runQuery(senderPublicKey []byte, contractName string, methodName string, args ...interface{}) (response *codec.RunQueryResponse, err error) {
	payload, err := h.client.CreateQuery(senderPublicKey, contractName, methodName, args...)
	if err != nil {
		return nil, err
	}
	response, err = h.client.SendQuery(payload)
	return
}

func (h *harness) getTransactionStatus(txId string) (response *codec.GetTransactionStatusResponse, err error) {
	response, err = h.client.GetTransactionStatus(txId)
	return
}

func (h *harness) getTransactionReceiptProof(txId string) (response *codec.GetTransactionReceiptProofResponse, err error) {
	response, err = h.client.GetTransactionReceiptProof(txId)
	return
}

func (h *harness) absoluteUrlFor(endpoint string) string {
	return getConfig().baseUrl + endpoint
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
	m := make(metrics)
	_ = json.Unmarshal(readBytes, &m)

	return m
}

func (h *harness) deployContractAndRequireSuccess(t *testing.T, keyPair *keys.Ed25519KeyPair, contractName string, contractBytes ...[]byte) {

	h.waitUntilTransactionPoolIsReady(t)

	dcExResult, dcTxStatus, dcErr := h.deployNativeContract(keyPair, contractName, contractBytes...)

	require.Nil(t, dcErr, "expected deploy contract to succeed")
	require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED, dcTxStatus, "expected deploy contract to succeed")
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS, dcExResult, "expected deploy contract to succeed")
}

func (h *harness) waitUntilTransactionPoolIsReady(t *testing.T) {

	recentBlockTimeDiff := getE2ETransactionPoolNodeSyncRejectTime() / 2
	require.True(t, test.Eventually(5*time.Second, func() bool {

		m := h.getMetrics()
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
		primitives.NodeAddress{},
		primitives.EcdsaSecp256K1PrivateKey{},
		nil,
		nil,
		"",
		"",
		"",
		primitives.NodeAddress{},
		0).
		TransactionPoolNodeSyncRejectTime()
}

func printTestTime(t *testing.T, msg string, last *time.Time) {
	t.Logf("%s (+%.3fs)", msg, time.Since(*last).Seconds())
	*last = time.Now()
}

func getConfig() E2EConfig {
	virtualChainId := uint32(42)

	if vcId, err := strconv.ParseUint(os.Getenv("VCHAIN"), 10, 0); err == nil {
		virtualChainId = uint32(vcId)
	}

	shouldBootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	baseUrl := fmt.Sprintf("http://localhost:%d", START_HTTP_PORT+2) // 8080 is leader, 8082 is node-3

	isRemoteEnvironment := os.Getenv("REMOTE_ENV") == "true"

	stressTestEnabled := os.Getenv("STRESS_TEST") == "true"
	stressTestNumberOfTransactions := int64(10000)
	stressTestFailureRate := int64(2)
	stressTestTargetTPS := float64(700)

	ethereumEndpoint := "http://127.0.0.1:8545"

	if !shouldBootstrap {
		apiEndpoint := os.Getenv("API_ENDPOINT")
		baseUrl = strings.TrimSuffix(strings.TrimRight(apiEndpoint, "/"), "/api/v1")
		ethereumEndpoint = os.Getenv("ETHEREUM_ENDPOINT")
	}

	if stressTestEnabled {
		stressTestNumberOfTransactions, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_NUMBER_OF_TRANSACTIONS"), 10, 0)
		stressTestFailureRate, _ = strconv.ParseInt(os.Getenv("STRESS_TEST_FAILURE_RATE"), 10, 0)
		stressTestTargetTPS, _ = strconv.ParseFloat(os.Getenv("STRESS_TEST_TARGET_TPS"), 0)
	}

	return E2EConfig{
		virtualChainId:    virtualChainId,
		bootstrap:         shouldBootstrap,
		remoteEnvironment: isRemoteEnvironment,
		baseUrl:           baseUrl,
		stressTest: StressTestConfig{
			enabled:               stressTestEnabled,
			numberOfTransactions:  stressTestNumberOfTransactions,
			acceptableFailureRate: stressTestFailureRate,
			targetTPS:             stressTestTargetTPS,
		},
		ethereumEndpoint: ethereumEndpoint,
	}
}
