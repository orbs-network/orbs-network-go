package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

//TODO: 2. create runnable in json api: orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --server-url=<http://....>
//TODO: 3. this test should use aforementioned runnable, sending the jsons as strings
//TODO: 4. move startSambusac into its own runnable main, taking --port=8080 argument
//TODO: 5. the sambusac server itself should run inside a docker container, as another runnable
func TestSambusacFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	port := ":8765"
	serverUrl := fmt.Sprintf("http://127.0.0.1%s", port)

	pathToContracts := "." //TODO compile contract(s) to SO, path points to dir containing them
	sambusac := jsonapi.StartSambusac(port, pathToContracts, false)
	defer sambusac.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	keyPair := keys.Ed25519KeyPairForTests(7)

	transferJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "transfer",
		Arguments: []jsonapi.MethodArgument{
			{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 42},
		},
	}

	sendTransactionOutput, err := jsonapi.SendTransaction(transferJson, keyPair, serverUrl)
	require.NoError(t, err, "error calling send_transfer")
	require.NotNil(t, sendTransactionOutput.TransactionReceipt.Txhash, "got empty txhash")

	time.Sleep(500 * time.Millisecond) //TODO remove when public api blocks on tx

	getBalanceJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	callMethodOutput, err := jsonapi.CallMethod(getBalanceJson, serverUrl)
	require.NoError(t, err, "error calling call_method")

	require.Len(t, callMethodOutput.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, 42, callMethodOutput.OutputArguments[0].Uint64Value, "expected balance to equal 42")
}
