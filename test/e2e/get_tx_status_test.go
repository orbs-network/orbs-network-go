package e2e

import (
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetTxStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	runMultipleTimes(t, func(t *testing.T) {

		h := NewAppHarness()
		lt := time.Now()
		PrintTestTime(t, "started", &lt)

		h.WaitUntilTransactionPoolIsReady(t)
		PrintTestTime(t, "first block committed", &lt)

		PrintTestTime(t, "send deploy - start", &lt)

		res, err := h.GetTransactionStatus("0xc0058950D1bDDe15D06C2d7354C3CB15daE02CFC6Bf5934B358D43def1Dfe1a0c420da72E541Bd6e")
		require.NoError(t, err, "expected polling for the status of an unsent transaction to return status without error")
		require.EqualValues(t, codec.TRANSACTION_STATUS_NO_RECORD_FOUND, res.TransactionStatus)
	})
}
