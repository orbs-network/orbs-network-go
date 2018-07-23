package instrumentation

import (
	"testing"
	"go.uber.org/zap"
)

const (
	TransactionFlow = "TransactionFlow"
)

const (
	TransactionAccepted = "Transaction accepted"
)

type BasicLogger interface {
	Info(flow string, message string, params... zap.Field)
}

type basicLogger struct {
	node string
	vchain int
	service string

	logger *zap.Logger
}

func (b *basicLogger) Info(flow string, message string, params... zap.Field){
	defer b.logger.Sync()

	zapParams := []zap.Field{
		zap.String("node", b.node),
		zap.Int("vchain", b.vchain),
		zap.String("flow", TransactionFlow),
	}

	zapParams = append(zapParams, params...)

	b.logger.Info(message, zapParams...)
}

func getLogger(node string, vchain int, service string) BasicLogger {
	zapLogger, _ := zap.NewProduction()
	logger := &basicLogger{node, vchain, service, zapLogger}

	return logger
}

func TestReporint(t *testing.T) {
	logger := getLogger("node1", 123, "public-api")

	txId := "1234567"

	logger.Info(TransactionFlow, TransactionAccepted, zap.String("txId", txId))
}