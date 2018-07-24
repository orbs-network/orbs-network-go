package instrumentation

import (
	"testing"
	"fmt"
	"runtime"
	"time"
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
)

const (
	TransactionFlow = "TransactionFlow"
)

const (
	TransactionAccepted = "Transaction accepted"
)

type BasicLogger interface {
	Info(flow string, message string, params... *Field)
}

type jsonLogger struct {

}

type basicLogger struct {
	node string
	vchain int
	service string

	jsonLogger *jsonLogger
}

type FieldType uint8

const (
	NoType = iota
	StringType
	IntType
	UintType
	BytesType
)

type Field struct {
	Key string
	Type FieldType

	String string
	Int int64
	Uint uint64
	Bytes []byte
}

func String(key string, value string) *Field {
	return &Field{Key: key, String: value, Type: StringType}
}

func Int(key string, value int) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Int32(key string, value int32) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Int64(key string, value int64) *Field {
	return &Field{Key: key, Int: int64(value), Type: IntType}
}

func Bytes(key string, value []byte) *Field {
	return &Field{Key: key, Bytes: value, Type: BytesType}
}

func Uint(key string, value uint) *Field {
	return &Field{Key: key, Uint: uint64(value), Type: UintType}
}

func Uint32(key string, value uint32) *Field {
	return &Field{Key: key, Uint: uint64(value), Type: UintType}
}

func Uint64(key string, value uint64) *Field {
	return &Field{Key: key, Uint: value, Type: UintType}
}

func getCaller() (function string, source string) {
	fpcs := make([]uintptr, 1)

	// skip levels to get to the caller of logger function
	n := runtime.Callers(5, fpcs)
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

func (j *jsonLogger) Log(level string, message string, params... *Field) {
	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine["timestamp"] = float64(time.Now().UTC().UnixNano()) / 1000000000
	logLine["message"] = message

	function, source := getCaller()
	logLine["function"] = function
	logLine["source"] = source

	for _, param := range params {
		switch param.Type {
		case StringType:
			logLine[param.Key] = param.String
		case IntType:
			logLine[param.Key] = param.Int
		case UintType:
			logLine[param.Key] = param.Uint
		case BytesType:
			logLine[param.Key] = base58.Encode(param.Bytes)
		}
	}

	logLineAsJson, _ := json.Marshal(logLine)

	fmt.Println(string(logLineAsJson))
}

func (j *jsonLogger) Info(message string, params... *Field) {
	j.Log("info", message, params...)
}

func (b *basicLogger) Info(flow string, message string, params... *Field){
	logLineParams := []*Field{
		String("node", b.node),
		Int("vchain", b.vchain),
		String("flow", TransactionFlow),
	}

	logLineParams = append(logLineParams, params...)
	b.jsonLogger.Info(message, logLineParams...)
}

func getLogger(node string, vchain int, service string) BasicLogger {
	logger := &basicLogger{node, vchain, service, &jsonLogger{}}

	return logger
}

func TestReporint(t *testing.T) {
	logger := getLogger("node1", 123, "public-api")

	logger.Info(TransactionFlow, TransactionAccepted, String("txId", "1234567"), Bytes("payload", []byte{1, 2, 3, 99}))
}