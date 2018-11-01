package test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/orbs-network/orbs-network-go/devtools/gammacli"
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

type harness struct {
	gamma                 *gammacli.GammaServer
	port                  int
}

func (h *harness) shutdown() {
	h.gamma.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
}

func newHarness() *harness {
	server := gammacli.StartGammaServer(":0", false)
	return &harness{gamma: server, port: server.Port()}
}

func (h *harness) cliBinaryPath() []string {
	ciCliBinaryPath := "/opt/orbs/gamma-cli"
	if _, err := os.Stat(ciCliBinaryPath); err == nil {
		return []string{ciCliBinaryPath}
	}

	if precompiledBinaryPath == "" { // only do this once per process so as to share compiled binary between tests of this package
		h.compileBinary()
	}

	return []string{precompiledBinaryPath}
}

var precompiledBinaryPath string
func (h *harness) compileBinary() {
	binaryDir, err := ioutil.TempDir("", "gamma")
	if err != nil {
		panic(err)
	}

	precompiledBinaryPath = binaryDir + "/gamma-cli"
	cmd := exec.Command("go", "build", "-o", precompiledBinaryPath, "../../gammacli/main/main.go")
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

func (h *harness) runCliCommand(t *testing.T, cliArgs ...string) string {
	h.gamma.Logger.Info("runCliCommand about to run command " + strings.Join(cliArgs, " "))
	defer h.gamma.Logger.Info("runCliCommand finished running command " + strings.Join(cliArgs, " "))

	command := h.cliBinaryPath()
	command = append(command, cliArgs...)

	cmd := exec.Command(command[0], command[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	require.NoError(t, err, "gamma cli command should not fail")

	s := stdout.String()
	return s
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

func getNodeUrl(port int) string {
	return "http://localhost:" + strconv.FormatUint(uint64(port), 10)
}

func (h *harness) transferAmountToAddress(t *testing.T, keyPair *keys.Ed25519KeyPair, targetAddress primitives.Ripmd160Sha256, amount uint64) {
	transferJSONBytes := generateTransferJSON(amount, targetAddress)

	err := ioutil.WriteFile("../json/transfer.json", transferJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	sendCommandOutput := h.runCliCommand(t, "run", "send", "../json/transfer.json",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String(), "-host", getNodeUrl(h.port))

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(sendCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.TransactionReceipt.ExecutionResult, "JSONTransaction status to be successful = 1")
	require.Equal(t, 1, response.TransactionStatus, "JSONTransaction status to be successful = 1")
	require.NotNil(t, response.TransactionReceipt.Txhash, "got empty txhash")
}

func (h *harness) getBalanceOfAddress(t *testing.T, targetAddress primitives.Ripmd160Sha256, expectedAmount uint64) {
	getBalanceJSONBytes := generateGetBalanceJSON(targetAddress)
	err := ioutil.WriteFile("../json/getBalance.json", getBalanceJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write getBalance JSON file")

	callOutputAsString := h.runCliCommand(t, "run", "call", "../json/getBalance.json", "-host", getNodeUrl(h.port))

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, expectedAmount, uint64(callResponse.OutputArguments[0].Value.(float64)), "expected balance to equal 42")
}

func (h *harness) deployCounterContract(t *testing.T, keyPair *keys.Ed25519KeyPair) {
	deployCommandOutput := h.runCliCommand(t, "deploy", "Counter", "../counterContract/counter.go",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String(), "-host", getNodeUrl(h.port))

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(deployCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.TransactionReceipt.ExecutionResult, "Transaction status to be successful = 1")
	require.Equal(t, 1, response.TransactionStatus, "Transaction status to be successful = 1")
	require.NotNil(t, response.TransactionReceipt.Txhash, "got empty txhash")
}

func (h *harness) getCounterValue(t *testing.T, expectedReturnValue uint64) {
	getCounterJSONBytes := generateGetCounterJSON()
	err := ioutil.WriteFile("../json/getCounter.json", getCounterJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	// Our contract is deployed, now let's continue to see we get 0 for the counter value (as it's the value it's init'd to
	callOutputAsString := h.runCliCommand(t, "run", "call", "../json/getCounter.json", "-host", getNodeUrl(h.port))

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, callResponse.CallResult, "wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from Counter.get()")
	require.EqualValues(t, expectedReturnValue, uint64(callResponse.OutputArguments[0].Value.(float64)), "counter value did not match expected one")
}

func (h *harness) addAmountToCounter(t *testing.T, keyPair *keys.Ed25519KeyPair, amount uint64) {
	addCounterJSONBytes := generateAddCounterJSON(amount)
	err := ioutil.WriteFile("../json/add.json", addCounterJSONBytes, 0644)
	if err != nil {
		t.Log("Couldn't write file", err)
	}
	require.NoError(t, err, "Couldn't write transfer JSON file")

	addOutputAsString := h.runCliCommand(t, "run", "send", "../json/add.json",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String(), "-host", getNodeUrl(h.port))

	addResponse := &sendTransactionCliResponse{}
	addResponseUnmarshalErr := json.Unmarshal([]byte(addOutputAsString), &addResponse)

	require.NoError(t, addResponseUnmarshalErr, "error calling Counter.add()")
	require.Equal(t, 1, addResponse.TransactionReceipt.ExecutionResult, "Wrong ExecutionResult value (expected 1 for success)")
	require.EqualValues(t, nil, addResponse.TransactionReceipt.OutputArguments, "expected no output arguments")
}

func TestGammaFlowWithActualJSONFilesUsingBenchmarkToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	h := newHarness()
	defer h.shutdown()

	keyPair := keys.Ed25519KeyPairForTests(0)
	targetAddress := builders.AddressForEd25519SignerForTests(2)
	var amount uint64 = 42

	h.transferAmountToAddress(t, keyPair, targetAddress, amount)
	h.getBalanceOfAddress(t, targetAddress, amount)
}

func TestGammaCliDeployWithUserDefinedContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	h := newHarness()
	defer h.shutdown()

	keyPair := keys.Ed25519KeyPairForTests(0)

	h.deployCounterContract(t, keyPair)
	h.getCounterValue(t, 0)

	// Add a random amount to the counter using Counter.add()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomAddAmount := uint64(r.Intn(4000)) + 1000 // Random int between 1000 and 5000

	h.addAmountToCounter(t, keyPair, randomAddAmount)

	h.getCounterValue(t, randomAddAmount)
}
