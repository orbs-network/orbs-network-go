package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/orbs-network/orbs-network-go/jsonapi"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
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

func ClientBinary() []string {
	ciBinaryPath := "/opt/orbs/orbs-json-client"
	if _, err := os.Stat(ciBinaryPath); err == nil {
		return []string{ciBinaryPath}
	}

	return []string{"go", "run", "../../jsonapi/main/json_client_cli.go"}
}

func TestSambusacFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	port := ":8080"
	pathToContracts := "." //TODO compile contract(s) to SO, path points to dir containing them
	sambusac := jsonapi.StartSambusac(port, pathToContracts, false)
	defer sambusac.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	transferJSON := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "transfer",
		Arguments: []jsonapi.MethodArgument{
			{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 42},
		},
	}

	jsonBytes, _ := json.Marshal(&transferJSON)

	baseCommand := ClientBinary()
	sendCommand := append(baseCommand, "-send-transaction", string(jsonBytes))

	cmd := exec.Command(sendCommand[0], sendCommand[1:]...)
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

	require.NoError(t, err, "error calling send_transfer")
	require.NoError(t, unmarshallErr, "error unmarshall cli response")
	require.Equal(t, response.ExecutionResult, 0, "Transaction status to be successful = 0")
	require.NotNil(t, response.TxHash, "got empty txhash")

	time.Sleep(500 * time.Millisecond) //TODO remove when public api blocks on tx

	getBalanceJSON := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	callJSONBytes, _ := json.Marshal(&getBalanceJSON)

	getCommand := append(baseCommand, "-call-method", string(callJSONBytes))

	callCmd := exec.Command(getCommand[0], getCommand[1:]...)
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

	require.NoError(t, callUnmarshallErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, 42, callResponse.OutputArguments[0].Uint64Value, "expected balance to equal 42")
}
