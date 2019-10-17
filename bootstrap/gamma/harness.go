package gamma

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const WAIT_FOR_BLOCK_TIMEOUT = 20 * time.Second

type metrics map[string]map[string]interface{}

func waitForBlock(endpoint string, targetBlockHeight primitives.BlockHeight) func() error {
	return func() error {
		res, err := http.Get(endpoint + "/metrics")
		if err != nil {
			return err
		}

		readBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		m := make(metrics)
		json.Unmarshal(readBytes, &m)

		blockHeight := m["BlockStorage.BlockHeight"]["Value"].(float64)
		if primitives.BlockHeight(blockHeight) < targetBlockHeight {
			return errors.Errorf("block %d is less than target block %d", int(blockHeight), targetBlockHeight)
		}

		return nil
	}
}

func RunOnRandomPort(t testing.TB, overrideConfig string) string {
	port := RunMain(t, -1, overrideConfig)
	endpoint := fmt.Sprintf("http://127.0.0.1:%d", port)
	t.Log(t.Name(), "started Gamma at", endpoint)
	require.NoError(t, test.RetryAndLog(WAIT_FOR_BLOCK_TIMEOUT, log.GetLogger(), waitForBlock(endpoint, 1)), "Gamma did not start ")

	return endpoint
}

func SendTransaction(t testing.TB, orbs *orbsClient.OrbsClient, sender *orbsClient.OrbsAccount, contractName string, method string, args ...interface{}) *codec.SendTransactionResponse {

	tx, _, err := orbs.CreateTransaction(sender.PublicKey, sender.PrivateKey, contractName, method, args...)
	require.NoError(t, err, "failed creating tx %s.%s", contractName, method)
	res, err := orbs.SendTransaction(tx)
	require.NoError(t, err, "failed sending tx %s.%s", contractName, method)
	require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED.String(), res.TransactionStatus.String(), "transaction to %s.%s not committed", contractName, method)
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "transaction to %s.%s not successful", contractName, method)

	return res
}

func DeployContract(t *testing.T, orbs *orbsClient.OrbsClient, account *orbsClient.OrbsAccount, contractName string, code []byte) {
	SendTransaction(t, orbs, account, "_Deployments", "deployService", "LogCalculator", uint32(protocol.PROCESSOR_TYPE_NATIVE), []byte(code))
}

func SendQuery(t testing.TB, orbs *orbsClient.OrbsClient, sender *orbsClient.OrbsAccount, minBlockHeight uint64, contractName string, method string, args ...interface{}) *codec.RunQueryResponse {
	q, err := orbs.CreateQuery(sender.PublicKey, contractName, method, args...)
	require.NoError(t, err, "failed creating query %s.%s", contractName, method)

	// Allow no more than 10 seconds for the network to sync
	var res *codec.RunQueryResponse
	require.True(t, test.Eventually(10*time.Second, func() bool {
		res, err = orbs.SendQuery(q)
		return err != nil || res.BlockHeight >= minBlockHeight
	}), "network did not sync - unable to obtain response for a lower bound block height")

	require.NoError(t, err, "failed sending query %s.%s", contractName, method)
	require.EqualValues(t, codec.REQUEST_STATUS_COMPLETED.String(), res.RequestStatus.String(), "failed calling %s.%s", contractName, method)
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "failed calling %s.%s", contractName, method)

	return res
}

func TimeTravel(t *testing.T, endpoint string, delta time.Duration) {
	res, err := http.Post(fmt.Sprintf("%s/debug/gamma/inc-time?seconds-to-add=%.0f", endpoint, delta.Seconds()), "text/plain", nil)
	require.NoError(t, err, "failed incrementing next block time")
	require.EqualValues(t, 200, res.StatusCode, "http call to increment time failed")
}

func Shutdown(t *testing.T, endpoint string) {
	res, err := http.Post(fmt.Sprintf("%s/debug/gamma/shutdown", endpoint), "text/plain", nil)
	require.NoError(t, err, "failed sending shutdown call")
	require.EqualValues(t, 200, res.StatusCode, "failed sending shutdown call")
}
