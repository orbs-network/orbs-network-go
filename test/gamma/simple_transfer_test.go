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

	cli := test.GammaCliWithPort(h.port)

	out, err := cli.Run("send-tx", "-i", "transfer.json")
	t.Log(out)
	require.NoError(t, err, "transfer should succeed")
	require.True(t, strings.Contains(out, `"ExecutionResult": "SUCCESS"`))

	txId := extractTxIdFromSendTxOutput(out)
	t.Log(txId)

	out, err = cli.Run("status", "-txid", txId)
	t.Log(out)
	require.NoError(t, err, "get tx status should succeed")
	require.True(t, strings.Contains(out, `"RequestStatus": "COMPLETED"`))

	out, err = cli.Run("read", "-i", "get-balance.json")
	t.Log(out)
	require.NoError(t, err, "get balance should succeed")
	require.True(t, strings.Contains(out, `"ExecutionResult": "SUCCESS"`))
	require.True(t, strings.Contains(out, `"Value": "17"`))
}
