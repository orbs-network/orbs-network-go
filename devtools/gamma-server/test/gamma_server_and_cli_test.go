package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func ClientBinary() []string {
	ciBinaryPath := "/opt/orbs/gamma-cli"
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

func TestGammaFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}

	port := ":8080"
	gamma := gammacli.StartGammaServer(port, false)
	defer gamma.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	keyPair := keys.Ed25519KeyPairForTests(0)

	baseCommand := ClientBinary()
	sendCommand := append(baseCommand,
		"run", "send", "../json/transfer.json",
		"-public-key", keyPair.PublicKey().String(),
		"-private-key", keyPair.PrivateKey().String())

	sendCommandOutput := runCommand(sendCommand, t)

	response := &sendTransactionCliResponse{}
	unmarshalErr := json.Unmarshal([]byte(sendCommandOutput), &response)

	require.NoError(t, unmarshalErr, "error unmarshal cli response")
	require.Equal(t, 1, response.TransactionReceipt.ExecutionResult, "JSONTransaction status to be successful = 1")
	require.Equal(t, 1, response.TransactionStatus, "JSONTransaction status to be successful = 1")
	require.NotNil(t, response.TransactionReceipt.Txhash, "got empty txhash")

	getCommand := append(baseCommand, "run", "call", "../json/getBalance.json")

	callOutputAsString := runCommand(getCommand, t)
	fmt.Println(callOutputAsString)

	callResponse := &callMethodCliResponse{}
	callUnmarshalErr := json.Unmarshal([]byte(callOutputAsString), &callResponse)

	require.NoError(t, callUnmarshalErr, "error calling call_method")
	require.Equal(t, 0, callResponse.CallResult, "Wrong callResult value")
	require.Len(t, callResponse.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, uint64(42), uint64(callResponse.OutputArguments[0].Value.(float64)), "expected balance to equal 42")
}
