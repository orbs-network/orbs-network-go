package test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/orbs-network/orbs-network-go/devtools/gammacli"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
)

type TransactionReceipt struct {
	Txhash          string
	ExecutionResult int
	OutputArguments interface{}
}

type sendTransactionCliResponse struct {
	TransactionReceipt TransactionReceipt
	TransactionStatus  int
	BlockHeight        int
	BlockTimestamp     int
}

type OutputArgumentCliResponse struct {
	Name  string
	Type  string
	Value interface{}
}

type callMethodCliResponse struct {
	OutputArguments []OutputArgumentCliResponse
	CallResult      int
	BlockHeight     int
	BlockTimestamp  int
}

func cliBinaryPath() []string {
	ciCliBinaryPath := "/opt/orbs/gamma-cli"
	if _, err := os.Stat(ciCliBinaryPath); err == nil {
		return []string{ciCliBinaryPath}
	}

	return []string{"go", "run", "../../gammacli/main/main.go"}
}

func runCliCommand(t *testing.T, cliArgs ...string) string {
	command := cliBinaryPath()
	command = append(command, cliArgs...)

	cmd := exec.Command(command[0], command[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	require.NoError(t, err, "gamma cli command should not fail")

	return stdout.String()
}

func generateTransferJSON(amount uint64, targetAddress []byte) []byte {
	// Encode the address as hex
	hexTargetAddress := hex.EncodeToString(targetAddress)

	transferJSON := &gammacli.JSONTransaction{
		ContractName: "BenchmarkToken",
		MethodName:   "transfer",
		Arguments: []gammacli.JSONMethodArgument{
			{Name: "amount", Type: "uint64", Value: amount},
			{Name: "targetAddress", Type: "bytes", Value: hexTargetAddress},
		},
	}

	jsonBytes, _ := json.Marshal(&transferJSON)
	return jsonBytes
}

func generateGetBalanceJSON(targetAddress []byte) []byte {
	// Encode the address as hex
	hexTargetAddress := hex.EncodeToString(targetAddress)

	getBalanceJSON := &gammacli.JSONTransaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
		Arguments: []gammacli.JSONMethodArgument{
			{Name: "targetAddress", Type: "bytes", Value: hexTargetAddress},
		},
	}

	callJSONBytes, _ := json.Marshal(&getBalanceJSON)
	return callJSONBytes
}

func generateGetCounterJSON() []byte {
	getCounterJSON := &gammacli.JSONTransaction{
		ContractName: "Counter",
		MethodName:   "get",
		Arguments:    []gammacli.JSONMethodArgument{},
	}

	callJSONBytes, _ := json.Marshal(&getCounterJSON)
	return callJSONBytes
}

func generateAddCounterJSON(amount uint64) []byte {
	arg := gammacli.JSONMethodArgument{Name: "amount", Type: "uint64", Value: amount}

	addAmountToCounterJSON := &gammacli.JSONTransaction{
		ContractName: "Counter",
		MethodName:   "add",
		Arguments:    []gammacli.JSONMethodArgument{arg},
	}

	addJSONBytes, _ := json.Marshal(&addAmountToCounterJSON)
	return addJSONBytes
}

func TestGammaFlowWithActualJSONFilesUsingBenchmarkToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	port := ":8080"
	gamma := gammacli.StartGammaServer(port, false)
	defer gamma.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	keyPair := keys.Ed25519KeyPairForTests(0)
	targetAddress := builders.AddressForEd25519SignerForTests(2)

	transferJSONBytes := generateTransferJSON(42, targetAddress)

	err := ioutil.WriteFile("../json/transfer.json", transferJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	sendCommandOutput := runCliCommand(t, "run", "send", "../json/transfer.json",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String())

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(sendCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.TransactionReceipt.ExecutionResult, "JSONTransaction status to be successful = 1")
	require.Equal(t, 1, response.TransactionStatus, "JSONTransaction status to be successful = 1")
	require.NotNil(t, response.TransactionReceipt.Txhash, "got empty txhash")

	getBalanceJSONBytes := generateGetBalanceJSON(targetAddress)
	err = ioutil.WriteFile("../json/getBalance.json", getBalanceJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write getBalance JSON file")

	callOutputAsString := runCliCommand(t, "run", "call", "../json/getBalance.json")

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, uint64(42), uint64(callResponse.OutputArguments[0].Value.(float64)), "expected balance to equal 42")
}

func TestGammaCliDeployWithUserDefinedContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	port := ":8080"
	gamma := gammacli.StartGammaServer(port, false)
	defer gamma.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	keyPair := keys.Ed25519KeyPairForTests(0)

	deployCommandOutput := runCliCommand(t, "deploy", "Counter", "../counterContract/counter.go",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String())

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(deployCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.TransactionReceipt.ExecutionResult, "Transaction status to be successful = 1")
	require.Equal(t, 1, response.TransactionStatus, "Transaction status to be successful = 1")
	require.NotNil(t, response.TransactionReceipt.Txhash, "got empty txhash")

	getCounterJSONBytes := generateGetCounterJSON()
	err := ioutil.WriteFile("../json/getCounter.json", getCounterJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	// Our contract is deployed, now let's continue to see we get 0 for the counter value (as it's the value it's init'd to
	callOutputAsString := runCliCommand(t, "run", "call", "../json/getCounter.json")

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from Counter.get()")
	require.EqualValues(t, uint64(0), uint64(callResponse.OutputArguments[0].Value.(float64)), "expected counter value to equal 0")

	// Add a random amount to the counter using Counter.add()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomAddAmount := uint64(r.Intn(4000)) + 1000 // Random int between 1000 and 5000

	addCounterJSONBytes := generateAddCounterJSON(randomAddAmount)
	err = ioutil.WriteFile("../json/add.json", addCounterJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	addOutputAsString := runCliCommand(t, "run", "send", "../json/add.json",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String())

	addResponse := &sendTransactionCliResponse{}
	addResponseUnmarshalErr := json.Unmarshal([]byte(addOutputAsString), &addResponse)

	require.NoError(t, addResponseUnmarshalErr, "error calling Counter.add()")
	require.Equal(t, 1, addResponse.TransactionReceipt.ExecutionResult, "Wrong ExecutionResult value (expected 1 for success)")
	require.EqualValues(t, nil, addResponse.TransactionReceipt.OutputArguments, "expected no output arguments")

	// Our contract is deployed, now let's continue to see we get 0 for the counter value (as it's the value it's init'd to
	callOutputSecondTimeAsString := runCliCommand(t, "run", "call", "../json/getCounter.json")

	callSecondResponse := &callMethodCliResponse{}
	err = json.Unmarshal([]byte(callOutputSecondTimeAsString), &callSecondResponse)

	require.NoError(t, err, "error calling Counter.get()")
	require.Equal(t, 0, callSecondResponse.CallResult, "Wrong callResult value")
	require.Len(t, callSecondResponse.OutputArguments, 1, "expected exactly one output argument returned from Counter.get()")
	require.EqualValues(t, randomAddAmount, uint64(callSecondResponse.OutputArguments[0].Value.(float64)), "expected counter value to equal our newly set value")
}
