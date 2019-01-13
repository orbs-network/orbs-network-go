package gamma

import (
	"github.com/orbs-network/orbs-client-sdk-go/gammacli/test"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSimpleTransfer(t *testing.T) {
	h := newHarness()
	defer h.shutdown()

	h.waitUntilTransactionPoolIsReady(t)

	cli := test.GammaCliWithPort(h.port)

	out, err := cli.Run("send-tx", "transfer.json")
	require.Contains(t, out, `"ExecutionResult": "SUCCESS"`)

	txId := extractTxIdFromSendTxOutput(out)
	t.Log(txId)

	sendTxOut, err := cli.Run("tx-status", txId)
	t.Log(sendTxOut)
	require.NoError(t, err, "get tx status should succeed")
	require.True(t, strings.Contains(sendTxOut, `"RequestStatus": "COMPLETED"`))

	sendTxOut, err = cli.Run("run-query", "get-balance.json")
	t.Log(sendTxOut)
	require.NoError(t, err, "get balance should succeed")
	require.True(t, strings.Contains(sendTxOut, `"ExecutionResult": "SUCCESS"`))
	require.True(t, strings.Contains(sendTxOut, `"Value": "17"`))
}
