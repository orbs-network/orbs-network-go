package e2e

import (
	"testing"
	"net/http"
	"io/ioutil"
	"fmt"
	"strings"
	"github.com/stretchr/testify/require"
	"time"
	"encoding/json"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"

	"context"
	"os"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"sync"
)

func TestSambusacFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
	}
	port := ":8765"
	serverUrl := fmt.Sprintf("http://127.0.0.1%s", port)

	pathToContracts := "." //TODO compile contract(s) to SO, path points to dir containing them
	sambusac := startSambusac(port, pathToContracts)
	defer sambusac.GracefulShutdown(1 * time.Second)

	time.Sleep(500 * time.Millisecond) // wait for server to start

	transferJson := `{
		"contractName": "BenchmarkToken",
		"methodName": "transfer",
		"arguments": [42]
	}`

	res, err := http.Post(serverUrl + "/api/send_transaction", "application/json", strings.NewReader(transferJson))
	require.NoError(t, err, "error calling send_transfer")
	require.Equal(t, http.StatusOK, res.StatusCode)

	time.Sleep(500 * time.Millisecond) //TODO remove when public api blocks on tx

	getBalanceJson := `{
		"contractName": "BenchmarkToken",
		"methodName": "getBalance",
	}`

	res, err = http.Post(serverUrl + "/api/cal_method", "application/json", strings.NewReader(getBalanceJson))
	require.NoError(t, err, "error calling call_method")
	require.Equal(t, http.StatusOK, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err, "error reading http response")

	type clientResponse struct {
		OutputArguments []interface{}
		CallResult protocol.ExecutionResult
		BlockHeight primitives.BlockHeight
		BlockTimestamp primitives.TimestampNano
	}

	var result clientResponse
	json.Unmarshal(bytes, &result)
	require.Len(t, result.OutputArguments, 1, "expected exactly one output argument returned from getBalance")
	balance, ok := result.OutputArguments[0].(uint64)
	require.True(t, ok, "expected uint64 returned from getBalance")
	require.Equal(t, 42, balance, "expected balance to equal 42")
}

type Sambusac struct {
	httpServer   httpserver.HttpServer
	logic        bootstrap.NodeLogic
	shutdownCond *sync.Cond
	ctxCancel    context.CancelFunc
}

func startSambusac(serverAddress string, pathToContracts string) *Sambusac {
	nodeKeyPair := keys.Ed25519KeyPairForTests(0)
	nodeName := fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

	federationNodes := make(map[string]config.FederationNode)
	publicKey := nodeKeyPair.PublicKey()
	federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	testLogger := instrumentation.GetLogger().WithOutput(instrumentation.NewOutput(os.Stdout).WithFormatter(instrumentation.NewHumanReadableFormatter()))

	config := config.NewHardCodedConfig(
		federationNodes,
		nodeKeyPair.PublicKey(),
		nodeKeyPair.PrivateKey(),
		nodeKeyPair.PublicKey(),
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		50, //TODO reduce to 1 milli
		70,
		5,
		5,
		30*60,
		5,
		3,
		300,
		300,
		1,
	)

	statePersistence := stateStorageAdapter.NewInMemoryStatePersistence()
	blockPersistence := blockStorageAdapter.NewInMemoryBlockPersistence()
	transport := gossipAdapter.NewTamperingTransport()
	ctx, cancel := context.WithCancel(context.Background())

	node := bootstrap.NewNodeLogic(
		ctx,
		transport,
		blockPersistence,
		statePersistence,
		testLogger.For(instrumentation.Node(nodeName)),
		config,
	)

	httpServer := httpserver.NewHttpServer(serverAddress, testLogger, node.PublicApi())

	s := &Sambusac{
		ctxCancel: cancel,
		shutdownCond: sync.NewCond(&sync.Mutex{}),
		logic: node,
		httpServer: httpServer,
	}

	go s.WaitUntilShutdown()

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




