package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"log"
	"os/exec"
	"testing"
	"time"
)

type sendTransactionCliResponse struct {
	TxHash          string
	ExecutionResult int
	OutputArguments []string
}

type OutputArgumentCliResponse struct {
	Name        string
	Type        int
	Uint32Value int32
	Uint64Value int64
	StringValue string
	BytesValue  []byte
}

type callMethodCliResponse struct {
	OutputArguments []OutputArgumentCliResponse
	CallResult      int
	BlockHeight     int
	BlockTimestamp  int
}

//TODO: 2. create runnable in json api: orbs-json-client [--send-transaction | --call-method]=<json> --public-key=<pubkey> --private-key=<privkey> --server-url=<http://....>
//TODO: 3. this test should use aforementioned runnable, sending the jsons as strings
//TODO: 4. move startSambusac into its own runnable main, taking --port=8080 argument
//TODO: 5. the sambusac server itself should run inside a docker container, as another runnable
func TestSambusacFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	port := ":8080"
	//serverUrl := fmt.Sprintf("http://127.0.0.1%s", port)
	pathToContracts := "." //TODO compile contract(s) to SO, path points to dir containing them
	sambusac := jsonapi.StartSambusac(port, pathToContracts, false)
	defer sambusac.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	//keyPair := keys.Ed25519KeyPairForTests(7)

	transferJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "transfer",
		Arguments: []jsonapi.MethodArgument{
			{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 42},
		},
	}

	jsonBytes, _ := json.Marshal(&transferJson)

	cmd := exec.Command("go", "run", "../../jsonapi/main/json_client_cli.go", "-send-transaction", string(jsonBytes))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	outputAsString := out.String()
	fmt.Println(outputAsString)

	response := &sendTransactionCliResponse{}
	unmarshallErr := json.Unmarshal([]byte(outputAsString), &response)

	///sendTransactionOutput, err := jsonapi.SendTransaction(transferJson, keyPair, serverUrl)
	require.NoError(t, err, "error calling send_transfer")
	require.NoError(t, unmarshallErr, "error unmarshall cli response")
	require.Equal(t, response.ExecutionResult, 0, "Transaction status to be successful = 0")
	require.NotNil(t, response.TxHash, "got empty txhash")

	time.Sleep(500 * time.Millisecond) //TODO remove when public api blocks on tx

	getBalanceJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	callJsonBytes, _ := json.Marshal(&getBalanceJson)

	callCmd := exec.Command("go", "run", "../../jsonapi/main/json_client_cli.go", "-call-method", string(callJsonBytes))
	var callOut bytes.Buffer
	callCmd.Stdout = &callOut
	callErr := callCmd.Run()
	if callErr != nil {
		log.Fatal(callErr)
	}

	callOutputAsString := callOut.String()
	fmt.Println(callOutputAsString)

	callResponse := &callMethodCliResponse{}
	callUnmarshallErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	//callMethodOutput, err := jsonapi.CallMethod(getBalanceJson, serverUrl)
	require.NoError(t, callUnmarshallErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, 42, callResponse.OutputArguments[0].Uint64Value, "expected balance to equal 42")
}
