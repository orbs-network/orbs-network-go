package e2e

import (
	"testing"
	"net/http"
	"io/ioutil"
	"fmt"
	"github.com/stretchr/testify/require"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"context"
	"os"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"sync"
	"github.com/orbs-network/orbs-network-go/jsonapi"
	"bytes"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/go-errors/errors"
	"github.com/orbs-network/orbs-network-go/test/harness"
)

var testLogger = instrumentation.GetLogger().WithOutput(instrumentation.NewOutput(os.Stdout).WithFormatter(instrumentation.NewHumanReadableFormatter()))

//TODO: 1. move sendTransactionJson and callMethodJson to jsonapi package (and omit the json suffix)
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
	sambusac := startSambusac(port, pathToContracts)
	defer sambusac.GracefulShutdown(1 * time.Second)

	time.Sleep(100 * time.Millisecond) // wait for server to start

	keyPair := keys.Ed25519KeyPairForTests(7)

	transferJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName: "transfer",
		Arguments: []jsonapi.MethodArgument{
			{Name:"amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 42},
		},
	}

	sendTransactionOutput, err := sendTransactionJson(transferJson, keyPair, serverUrl)
	require.NoError(t, err, "error calling send_transfer")
	require.NotNil(t, sendTransactionOutput.TransactionReceipt.Txhash, "got empty txhash")

	time.Sleep(500 * time.Millisecond) //TODO remove when public api blocks on tx

	getBalanceJson := &jsonapi.Transaction{
		ContractName: "BenchmarkToken",
		MethodName:   "getBalance",
	}

	callMethodOutput, err := CallMethodJson(getBalanceJson, serverUrl)
	require.NoError(t, err, "error calling call_method")

	require.Len(t, callMethodOutput.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	require.EqualValues(t, 42, callMethodOutput.OutputArguments[0].Uint64Value, "expected balance to equal 42")
}

func sendTransactionJson(transferJson *jsonapi.Transaction, keyPair *keys.Ed25519KeyPair, serverUrl string) (*jsonapi.SendTransactionOutput, error) {
	tx, err := jsonapi.ConvertAndSignTransaction(transferJson, keyPair)
	testLogger.Info("sending transaction", instrumentation.Stringable("transaction", tx.Build()))
	sendTransactionRequest := (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build()
	res, err := http.Post(serverUrl+"/api/send-transaction", "application/octet-stream", bytes.NewReader(sendTransactionRequest.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got unexpected http status code %s", res.StatusCode)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return jsonapi.ConvertSendTransactionOutput(client.SendTransactionResponseReader(bytes)), err
}

func CallMethodJson(transferJson *jsonapi.Transaction, serverUrl string) (*jsonapi.CallMethodOutput, error) {
	tx := jsonapi.ConvertTransaction(transferJson)
	testLogger.Info("calling method", instrumentation.Stringable("transaction", tx.Build()))
	request := (&client.CallMethodRequestBuilder{Transaction: tx}).Build()
	res, err := http.Post(serverUrl+"/api/call-method", "application/octet-stream", bytes.NewReader(request.Raw()))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got unexpected http status code %s", res.StatusCode)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	return jsonapi.ConvertCallMethodOutput(client.CallMethodResponseReader(bytes)), err

}

type Sambusac struct {
	httpServer   httpserver.HttpServer
	logic        bootstrap.NodeLogic
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
}

func startSambusac(serverAddress string, pathToContracts string) *Sambusac {
	ctx, cancel := context.WithCancel(context.Background())

	network := harness.NewTestNetwork(ctx, 3, consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS)

	httpServer := httpserver.NewHttpServer(serverAddress, testLogger, network.PublicApi(0))

	s := &Sambusac{
		ctxCancel: cancel,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		httpServer: httpServer,
	}

	go s.WaitUntilShutdown() //TODO remove 'go' and block

	return s
}

func (n *Sambusac) GracefulShutdown(timeout time.Duration) {
	n.ctxCancel()
	n.httpServer.GracefulShutdown(timeout)
	n.shutdownCond.Broadcast()
}

func (n *Sambusac) WaitUntilShutdown() {
	n.shutdownCond.L.Lock()
	n.shutdownCond.Wait()
	n.shutdownCond.L.Unlock()
}




