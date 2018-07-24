package instrumentation

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"io"
	"os"
	"runtime"
	"time"
)

const NanosecondsInASecond = 1000000000

type BasicLogger interface {
	Log(level string, message string, params ...*Field)
	Info(message string, params ...*Field)
	Metric(name string, value *Field)
	For(params ...*Field) BasicLogger
	Meter(name string, params ...*Field) BasicMeter
	Prefixes() []*Field
	WithOutput(writer io.Writer) BasicLogger
}

type basicLogger struct {
	output       io.Writer
	prefixes     []*Field
	nestingLevel int
}

type FieldType uint8

const (
	NoType = iota
	NodeType
	ServiceType
	StringType
	IntType
	UintType
	BytesType
	FloatType
)

type Field struct {
	Key  string
	Type FieldType

	String string
	Int    int64
	Uint   uint64
	Bytes  []byte

	Float float64
}

func Node(value string) *Field {
	return &Field{Key: "node", String: value, Type: NodeType}
}

func Service(value string) *Field {
	return &Field{Key: "service", String: value, Type: ServiceType}
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

func Float32(key string, value float32) *Field {
	return &Field{Key: key, Float: float64(value), Type: FloatType}
}

func Float64(key string, value float64) *Field {
	return &Field{Key: key, Float: value, Type: FloatType}
}

func getCaller(level int) (function string, source string) {
	fpcs := make([]uintptr, 1)

	// skip levels to get to the caller of logger function
	n := runtime.Callers(level, fpcs)
	if n == 0 {
		return "n/a", "n/a"
	}

	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a", "n/a"
	}

	file, line := fun.FileLine(fpcs[0] - 1)
	return fun.Name(), fmt.Sprintf("%s:%d", file, line)
}

func GetLogger(params ...*Field) BasicLogger {
	logger := &basicLogger{prefixes: params, nestingLevel: 4, output: os.Stdout}

	return logger
}

func (b *basicLogger) Prefixes() []*Field {
	return b.prefixes
}

func (b *basicLogger) For(params ...*Field) BasicLogger {
	prefixes := append(b.prefixes, params...)
	return &basicLogger{prefixes: prefixes, nestingLevel: b.nestingLevel, output: b.output}
}

func (b *basicLogger) Metric(metric string, value *Field) {
	b.Log("metric", "Metric recorded", String("metric", metric), value)
}

func (f *Field) Value() interface{} {
	switch f.Type {
	case NodeType:
		return f.String
	case ServiceType:
		return f.String
	case StringType:
		return f.String
	case IntType:
		return f.Int
	case UintType:
		return f.Uint
	case BytesType:
		return base58.Encode(f.Bytes)
	case FloatType:
		return f.Float
	}

	return nil
}

func (b *basicLogger) Log(level string, message string, params ...*Field) {
	params = append(params, b.prefixes...)

	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine["timestamp"] = float64(time.Now().UTC().UnixNano()) / NanosecondsInASecond
	logLine["message"] = message

	function, source := getCaller(b.nestingLevel)
	logLine["function"] = function
	logLine["source"] = source

	for _, param := range params {
		logLine[param.Key] = param.Value()
	}

	logLineAsJson, _ := json.Marshal(logLine)

	fmt.Fprintln(b.output, string(logLineAsJson))
}

func (b *basicLogger) Info(message string, params ...*Field) {
	b.Log("info", message, params...)
}

func (b *basicLogger) Meter(name string, params ...*Field) BasicMeter {
	meterLogger := &basicLogger{nestingLevel: 5, prefixes: b.prefixes, output: b.output}
	return &basicMeter{name: name, start: time.Now().UnixNano(), logger: meterLogger, params: params}
}

func (b *basicLogger) WithOutput(writer io.Writer) BasicLogger {
	b.output = writer
	return b
}
