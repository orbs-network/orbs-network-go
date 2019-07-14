package e2e

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const WAIT_FOR_BLOCK_TIMEOUT = 10 * time.Second

type metrics map[string]map[string]interface{}

func waitForBlock(endpoint string, targetBlockHeight primitives.BlockHeight) func() bool {
	return func() bool {
		res, err := http.Get(endpoint + "/metrics")
		if err != nil {
			fmt.Println(err)
			return false
		}

		readBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return false
		}
		m := make(metrics)
		json.Unmarshal(readBytes, &m)

		blockHeight := m["BlockStorage.BlockHeight"]["Value"].(float64)
		return primitives.BlockHeight(blockHeight) >= targetBlockHeight
	}
}

func sendTransaction(t testing.TB, orbs *orbsClient.OrbsClient, sender *orbsClient.OrbsAccount, contractName string, method string, args ...interface{}) *codec.SendTransactionResponse {

	tx, _, err := orbs.CreateTransaction(sender.PublicKey, sender.PrivateKey, contractName, method, args...)
	require.NoError(t, err, "failed creating tx %s.%s", contractName, method)
	res, err := orbs.SendTransaction(tx)
	require.NoError(t, err, "failed sending tx %s.%s", contractName, method)
	require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED.String(), res.TransactionStatus.String(), "transaction to %s.%s not committed", contractName, method)
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "transaction to %s.%s not successful", contractName, method)

	return res
}

func deployContract(t *testing.T, orbs *orbsClient.OrbsClient, account *orbsClient.OrbsAccount, contractName string, code []byte) {
	sendTransaction(t, orbs, account, "_Deployments", "deployService", "LogCalculator", uint32(protocol.PROCESSOR_TYPE_NATIVE), []byte(code))
}

func sendQuery(t testing.TB, orbs *orbsClient.OrbsClient, sender *orbsClient.OrbsAccount, contractName string, method string, args ...interface{}) *codec.RunQueryResponse {
	q, err := orbs.CreateQuery(sender.PublicKey, contractName, method, args...)
	require.NoError(t, err, "failed creating query %s.%s", contractName, method)
	res, err := orbs.SendQuery(q)
	require.NoError(t, err, "failed sending query %s.%s", contractName, method)
	require.EqualValues(t, codec.REQUEST_STATUS_COMPLETED.String(), res.RequestStatus.String(), "failed calling %s.%s", contractName, method)
	require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "failed calling %s.%s", contractName, method)

	return res
}
