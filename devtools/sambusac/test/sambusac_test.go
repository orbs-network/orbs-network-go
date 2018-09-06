package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
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

	return []string{"go", "run", "../../jsonapi/main/main.go"}
}

func runCommand(command []string, t *testing.T) string {
	cmd := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	fmt.Println("jsonapi exec command:", command)
	fmt.Println("command stdout:", stdout.String())
	fmt.Println("command stderr:", stderr.String())

	require.NoError(t, err, "jsonapi cli command should not fail")

	return stdout.String()
}

func generateTransferJSON() string {
	transferJSON := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "transfer",
		Arguments: []jsonapi.MethodArgument{
			{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 42},
		},
	}

	jsonBytes, _ := json.Marshal(&transferJSON)
	return string(jsonBytes)
}

func generateGetBalanceJSON() string {
	getBalanceJSON := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	callJSONBytes, _ := json.Marshal(&getBalanceJSON)
	return string(callJSONBytes)
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

	keyPair := keys.Ed25519KeyPairForTests(0)

	baseCommand := ClientBinary()
	sendCommand := append(baseCommand,
		"-send-transaction", generateTransferJSON(),
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String())

	sendCommandOutput := runCommand(sendCommand, t)

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(sendCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.ExecutionResult, "Transaction status to be successful = 1")
	require.NotNil(t, response.TxHash, "got empty txhash")

	getCommand := append(baseCommand, "-call-method", generateGetBalanceJSON())

	callOutputAsString := runCommand(getCommand, t)
	fmt.Println(callOutputAsString)

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, 42, callResponse.OutputArguments[0].Uint64Value, "expected balance to equal 42")
}
