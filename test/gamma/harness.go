package gamma

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
	"time"
)

type harness struct {
	gamma *gamma.GammaServer
	port  int
}

func (h *harness) shutdown() {
	h.gamma.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
}

func newHarness() *harness {
	server := gamma.StartGammaServer(":0", false)
	return &harness{gamma: server, port: server.Port()}
}

func extractTxIdFromSendTxOutput(out string) string {
	re := regexp.MustCompile(`\"TxId\":\s+\"(\w+)\"`)
	res := re.FindStringSubmatch(out)
	return res[1]
}

func (h *harness) waitUntilTransactionPoolIsReady(t *testing.T) {
	require.True(t, test.Eventually(3*time.Second, func() bool { // 3 seconds to avoid jitter but it really shouldn't take that long
		m := h.getMetrics()
		if m == nil {
			return false
		}

		blockHeight := m["TransactionPool.BlockHeight"]["Value"].(float64)

		return blockHeight > 0
	}), "Timed out waiting for metric TransactionPool.BlockHeight > 0")
}

type metrics map[string]map[string]interface{}

func (h *harness) getMetrics() metrics {
	res, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", h.port))

	if err != nil {
		panic(err)
	}

	if res == nil {
		return nil
	}

	readBytes, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(readBytes))

	m := make(metrics)
	json.Unmarshal(readBytes, &m)

	return m
}
