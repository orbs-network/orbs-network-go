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

		res, err := h.GetTransactionStatus("0xC0058950d1Bdde15d06C2d7354C3Cb15Dae02CFC6BF5934b358D43dEf1DFE1a0C420Da72e541bd6e")
		require.NoError(t, err)
		require.EqualValues(t, codec.TRANSACTION_STATUS_NO_RECORD_FOUND, res.TransactionStatus)
	})
}
