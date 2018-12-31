package gamma

import (
	"github.com/orbs-network/orbs-client-sdk-go/gammacli/test"
	testUtils "github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestSimpleTransfer(t *testing.T) {
	h := newHarness()
	defer h.shutdown()

	cli := test.GammaCliWithPort(h.port)

	sendTxOutChan := make(chan string, 1)

	// TODO remove Eventually loop once node can handle requests at block height 0
	require.True(t, testUtils.Eventually(1*time.Second, func() bool {
		out, err := cli.Run("send-tx", "transfer.json")
		t.Log(out)
		success := err == nil && strings.Contains(out, `"ExecutionResult": "SUCCESS"`)
		if success {
			sendTxOutChan <- out
		}
		return success
	}), "transfer should eventually succeed")

	sendTxOut := <-sendTxOutChan
	txId := extractTxIdFromSendTxOutput(sendTxOut)
	t.Log(txId)

	sendTxOut, err := cli.Run("get-status", txId)
	t.Log(sendTxOut)
	require.NoError(t, err, "get tx status should succeed")
	require.True(t, strings.Contains(sendTxOut, `"RequestStatus": "COMPLETED"`))

	sendTxOut, err = cli.Run("run-query", "get-balance.json")
	t.Log(sendTxOut)
	require.NoError(t, err, "get balance should succeed")
	require.True(t, strings.Contains(sendTxOut, `"ExecutionResult": "SUCCESS"`))
	require.True(t, strings.Contains(sendTxOut, `"Value": "17"`))
}
