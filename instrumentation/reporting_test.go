package instrumentation

import (
	"testing"
	"go.uber.org/zap"
	"time"
	"encoding/json"
	"fmt"
	"go.uber.org/zap/zapcore"
	"runtime"
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

type jsonLogger struct {

}

type basicLogger struct {
	node string
	vchain int
	service string

	logger *zap.Logger
	jsonLogger *jsonLogger
}

func getCaller() (function string, source string) {
	fpcs := make([]uintptr, 1)

	// skip 4 levels to get to the caller of logger function
	n := runtime.Callers(4, fpcs)
	if n == 0 {
		return "n/a", "n/a"
	}

	fun := runtime.FuncForPC(fpcs[0]-1)
	if fun == nil {
		return "n/a", "n/a"
	}

	file, line := fun.FileLine(fpcs[0]-1)
	return fun.Name(), fmt.Sprintf("%s:%d", file, line)
}

func (j *jsonLogger) Info(message string, params... zap.Field) {
	logLine := make(map[string]interface{})

	logLine["level"] = "info"
	logLine["timestamp"] = float64(time.Now().UTC().UnixNano()) / 1000000000
	logLine["message"] = message

	function, source := getCaller()
	logLine["function"] = function
	logLine["source"] = source

	for _, param := range params {
		switch param.Type {
		case zapcore.StringType:
			logLine[param.Key] = param.String
		case zapcore.Int64Type:
			logLine[param.Key] = param.Integer
		}

	}

	logLineAsJson, _ := json.Marshal(logLine)

	fmt.Println(string(logLineAsJson))
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
	b.jsonLogger.Info(message, zapParams...)
}

func getLogger(node string, vchain int, service string) BasicLogger {
	zapLogger, _ := zap.NewProduction()
	logger := &basicLogger{node, vchain, service, zapLogger, &jsonLogger{}}

	return logger
}

func TestReporint(t *testing.T) {
	logger := getLogger("node1", 123, "public-api")

	txId := "1234567"

	logger.Info(TransactionFlow, TransactionAccepted, zap.String("txId", txId))
}