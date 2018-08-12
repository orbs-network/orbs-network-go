package instrumentation

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

type BasicLogger interface {
	Log(level string, message string, params ...*Field)
	Info(message string, params ...*Field)
	Error(message string, params ...*Field)
	Metric(name string, params ...*Field)
	For(params ...*Field) BasicLogger
	Meter(name string, params ...*Field) BasicMeter
	Prefixes() []*Field
	WithOutput(writer io.Writer) BasicLogger
	WithFormatter(formatter LogFormatter) BasicLogger
}

type basicLogger struct {
	output                io.Writer
	formatter             LogFormatter
	prefixes              []*Field
	nestingLevel          int
	sourceRootPrefixIndex int
}

type FieldType uint8

const (
	NoType = iota
	ErrorType
	NodeType
	ServiceType
	StringType
	IntType
	UintType
	BytesType
	FloatType
	FunctionType
	SourceType
)

type Field struct {
	Key  string
	Type FieldType

	String string
	Int    int64
	Uint   uint64
	Bytes  []byte

	Float float64

	Error error
}

func Node(value string) *Field {
	return &Field{Key: "node", String: value, Type: NodeType}
}

func Service(value string) *Field {
	return &Field{Key: "service", String: value, Type: ServiceType}
}

func Function(value string) *Field {
	return &Field{Key: "function", String: value, Type: FunctionType}
}

func Source(value string) *Field {
	return &Field{Key: "source", String: value, Type: SourceType}
}

func String(key string, value string) *Field {
	return &Field{Key: key, String: value, Type: StringType}
}

func Stringable(key string, value fmt.Stringer) *Field {
	return &Field{Key: key, String: value.String(), Type: StringType}
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

func Error(value error) *Field {
	return &Field{Key: "error", Error: value, Type: ErrorType}
}

func BlockHeight(value primitives.BlockHeight) *Field {
	return &Field{Key: "blockHeight", String: value.String(), Type: StringType}
}

func GetLogger(params ...*Field) BasicLogger {
	logger := &basicLogger{prefixes: params, nestingLevel: 4, output: os.Stdout, formatter: NewJsonFormatter()}

	fpcs := make([]uintptr, 1)
	n := runtime.Callers(logger.nestingLevel, fpcs)
	if n != 0 {
		fun := runtime.FuncForPC(fpcs[0] - 1)
		if fun != nil {
			file, _ := fun.FileLine(fpcs[0] - 1)
			if l := strings.Index(file, "orbs-network-go/"); l > -1 {
				logger.sourceRootPrefixIndex = l + len("orbs-network-go/")
			}
		}
	}

	return logger
}

func (b *basicLogger) getCaller(level int) (function string, source string) {
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
	fName := fun.Name()
	lastSlashOfName := strings.LastIndex(fName, "/")
	if lastSlashOfName > 0 {
		fName = fName[lastSlashOfName+1:]
	}
	return fName, fmt.Sprintf("%s:%d", file[b.sourceRootPrefixIndex:], line)
}

func (b *basicLogger) Prefixes() []*Field {
	return b.prefixes
}

func (b *basicLogger) For(params ...*Field) BasicLogger {
	prefixes := append(b.prefixes, params...)
	return &basicLogger{prefixes: prefixes, nestingLevel: b.nestingLevel, output: b.output, formatter: b.formatter}
}

func (b *basicLogger) Metric(metric string, params ...*Field) {
	metricParams := append(params, String("metric", metric))
	b.Log("metric", "Metric recorded", metricParams...)
}

func (f *Field) Value() interface{} {
	switch f.Type {
	case NodeType:
		return f.String
	case ServiceType:
		return f.String
	case FunctionType:
		return f.String
	case SourceType:
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
	case ErrorType:
		return f.Error.Error()
	}

	return nil
}

func (b *basicLogger) Log(level string, message string, params ...*Field) {
	function, source := b.getCaller(b.nestingLevel)

	enrichmentParams := []*Field{
		Function(function),
		Source(source),
	}

	enrichmentParams = append(enrichmentParams, b.prefixes...)
	enrichmentParams = append(enrichmentParams, params...)

	logLine := b.formatter.FormatRow(level, message, enrichmentParams...)
	fmt.Fprintln(b.output, logLine)
}

func (b *basicLogger) Info(message string, params ...*Field) {
	b.Log("info", message, params...)
}

func (b *basicLogger) Error(message string, params ...*Field) {
	b.Log("info", message, params...)
}

func (b *basicLogger) Meter(name string, params ...*Field) BasicMeter {
	meterLogger := &basicLogger{nestingLevel: 5, prefixes: b.prefixes, output: b.output, formatter: b.formatter}
	return &basicMeter{name: name, start: time.Now().UnixNano(), logger: meterLogger, params: params}
}

func (b *basicLogger) WithOutput(writer io.Writer) BasicLogger {
	b.output = writer
	return b
}

func (b *basicLogger) WithFormatter(formatter LogFormatter) BasicLogger {
	b.formatter = formatter
	return b
}
