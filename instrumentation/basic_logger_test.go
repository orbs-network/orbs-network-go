package instrumentation

import (
	"testing"
	"time"
)

const (
	TransactionFlow = "TransactionFlow"
)

const (
	TransactionAccepted  = "Transaction accepted"
	TransactionProcessed = "Transaction processed"
)

func TestReport(t *testing.T) {
	serviceLogger := GetLogger(Node("node1"), Service("public-api"))
	serviceLogger.Info("Service initialized")

	txId := String("txId", "1234567")

	txFlowLogger := serviceLogger.For(String("flow", TransactionFlow))
	txFlowLogger.Info(TransactionAccepted, txId, Bytes("payload", []byte{1, 2, 3, 99}))

	prefixedLogger := txFlowLogger.For(String("one-more-prefix", "one-more-value"))

	prefixedLogger.Info(TransactionProcessed, txId)

	meter := txFlowLogger.Meter("tx-process-time", txId)
	defer meter.Done()

	time.Sleep(1 * time.Millisecond)
}
