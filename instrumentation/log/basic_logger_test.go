package log_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

const (
	TransactionFlow     = "TransactionFlow"
	TransactionAccepted = "Transaction accepted"
)

func captureStdout(f func(writer io.Writer)) string {
	r, w, _ := os.Pipe()

	f(w)

	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

type discardWriter struct {
}

func (w *discardWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func discardStdout(f func(writer io.Writer)) {
	writer := &discardWriter{}
	f(writer)
}

func parseOutput(input string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(input), &jsonMap)
	return jsonMap
}

func TestSimpleLogger(t *testing.T) {
	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer))
		serviceLogger.Info("Service initialized")
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestSimpleLogger.func1", jsonMap["function"])
	require.Equal(t, "Service initialized", jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
}

func TestBasicLogger_WithFilter(t *testing.T) {
	b := new(bytes.Buffer)
	log.GetLogger().WithOutput(log.NewOutput(b)).
		WithFilters(log.OnlyErrors()).
		Info("foo")
	require.Empty(t, b.String(), "output was not empty")
}

func TestCompareLogger(t *testing.T) {
	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer))
		serviceLogger.LogFailedExpectation("Service initialized compare", log.BlockHeight(primitives.BlockHeight(9999)), log.BlockHeight(primitives.BlockHeight(8888)), log.Bytes("bytes", []byte{2, 3, 99}))
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	require.Equal(t, "expectation", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestCompareLogger.func1", jsonMap["function"])
	require.Equal(t, "Service initialized compare", jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
	require.Equal(t, "020363", jsonMap["bytes"])
	require.Equal(t, "22b8", jsonMap["actual-block-height"])
	require.Equal(t, "270f", jsonMap["expected-block-height"])
}

func TestNestedLogger(t *testing.T) {
	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer))
		txId := log.String("txId", "1234567")
		txFlowLogger := serviceLogger.WithTags(log.String("flow", TransactionFlow))
		txFlowLogger.Info(TransactionAccepted, txId, log.Bytes("payload", []byte{1, 2, 3, 99, 250}))
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestNestedLogger.func1", jsonMap["function"])
	require.Equal(t, TransactionAccepted, jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
	require.Equal(t, "1234567", jsonMap["txId"])
	require.Equal(t, TransactionFlow, jsonMap["flow"])
	require.Equal(t, "01020363fa", jsonMap["payload"])

}

func TestStringableSlice(t *testing.T) {
	var receipts []*protocol.TransactionReceipt

	receipts = append(receipts, builders.TransactionReceipt().Build())
	receipts = append(receipts, builders.TransactionReceipt().Build())

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer))
		serviceLogger.Info("StringableSlice test", log.StringableSlice("a-collection", receipts))
	})

	fmt.Println(stdout)
	jsonMap := parseOutput(stdout)

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestStringableSlice.func1", jsonMap["function"])
	require.Equal(t, "StringableSlice test", jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
	require.NotEqual(t, "[]", jsonMap["a-collection"])

	require.Equal(t, []interface{}{
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,}",
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,}",
	}, jsonMap["a-collection"])
}

func TestStringableSliceCustomFormat(t *testing.T) {
	var transactions []*protocol.SignedTransaction

	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())
	transactions = append(transactions, builders.TransferTransaction().Build())

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer).WithFormatter(log.NewHumanReadableFormatter()))
		serviceLogger.Info("StringableSlice HR test", log.StringableSlice("a-collection", transactions))
	})

	fmt.Println(stdout)

	require.Regexp(t, "^info", stdout)
	require.Regexp(t, "StringableSlice HR test", stdout)
	require.Regexp(t, "node=node1", stdout)
	require.Regexp(t, "service=public-api", stdout)
	require.Regexp(t, "a-collection=", stdout)
	require.Regexp(t, "{Transaction:{ProtocolVersion:1,", stdout)
	require.Regexp(t, "function=log_test.TestStringableSliceCustomFormat.func1 ", stdout)
	require.Regexp(t, "source=instrumentation/log/basic_logger_test.go", stdout)
}

func TestMeter(t *testing.T) {
	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer))
		txId := log.String("txId", "1234567")
		txFlowLogger := serviceLogger.WithTags(log.String("flow", TransactionFlow))
		meter := txFlowLogger.Meter("tx-process-time", txId)
		defer meter.Done()

		time.Sleep(1 * time.Millisecond)
	})

	fmt.Println(stdout)

	jsonMap := parseOutput(stdout)

	require.Equal(t, "metric", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestMeter.func1", jsonMap["function"])
	require.Equal(t, "Metric recorded", jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
	require.Equal(t, "public-api-TransactionFlow-tx-process-time", jsonMap["metric"])
	require.Equal(t, "1234567", jsonMap["txId"])
	require.Equal(t, TransactionFlow, jsonMap["flow"])
	require.NotNil(t, jsonMap["process-time"])
}

func TestCustomLogFormatter(t *testing.T) {
	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer).WithFormatter(log.NewHumanReadableFormatter()))
		serviceLogger.Info("Service initialized", log.Int("some-int-value", 12), log.BlockHeight(primitives.BlockHeight(9999)), log.Bytes("bytes", []byte{2, 3, 99}), log.Stringable("vchainId", primitives.VirtualChainId(123)), log.String("_test-id", "hello"), log.String("_underscore", "wow"))
	})

	fmt.Println(stdout)

	require.Regexp(t, "^info", stdout)
	require.Regexp(t, "Service initialized", stdout)
	require.Regexp(t, "node=node1", stdout)
	require.Regexp(t, "service=public-api", stdout)
	require.Regexp(t, "block-height=270f", stdout)
	require.Regexp(t, "vchainId=7b", stdout)
	require.Regexp(t, "bytes=gDp", stdout)
	require.Regexp(t, "some-int-value=12", stdout)
	require.Regexp(t, "function=log_test.TestCustomLogFormatter.func1", stdout)
	require.Regexp(t, "source=instrumentation/log/basic_logger_test.go", stdout)
	require.Regexp(t, "_test-id=hello", stdout)
	require.Regexp(t, "_underscore=wow", stdout)
}

func TestMultipleOutputs(t *testing.T) {
	filename := "/tmp/test-multiple-outputs"
	os.RemoveAll(filename)

	fileOutput, _ := os.Create(filename)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer), log.NewOutput(fileOutput))
		serviceLogger.Info("Service initialized")
	})

	rawFile, _ := ioutil.ReadFile(filename)
	fileContents := string(rawFile)

	fmt.Println(fileContents)

	checkOutput(t, stdout)
	checkOutput(t, fileContents)
}

func TestMultipleOutputsForMemoryViolationByHumanReadable(t *testing.T) {
	filename := "/tmp/test-multiple-outputs"
	os.RemoveAll(filename)

	fileOutput, _ := os.Create(filename)

	require.NotPanics(t, func() {
		captureStdout(func(writer io.Writer) {
			serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(writer).WithFormatter(log.NewHumanReadableFormatter()), log.NewOutput(fileOutput))
			serviceLogger.Info("Service initialized")
		})
	})
}

func BenchmarkBasicLoggerInfoFormatters(b *testing.B) {
	receipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().WithRandomHash().Build(),
		builders.TransactionReceipt().WithRandomHash().Build(),
	}

	formatters := []log.LogFormatter{log.NewHumanReadableFormatter(), log.NewJsonFormatter()}

	discardStdout(func(writer io.Writer) {
		for _, formatter := range formatters {
			b.Run(reflect.TypeOf(formatter).String(), func(b *testing.B) {
				serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
					WithOutput(log.NewOutput(writer).WithFormatter(formatter))

				b.StartTimer()
				for i := 0; i < b.N; i++ {
					serviceLogger.Info("Benchmark test", log.StringableSlice("a-collection", receipts))
				}
				b.StopTimer()
			})
		}
	})
}

func BenchmarkBasicLoggerInfoWithDevNull(b *testing.B) {
	receipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().WithRandomHash().Build(),
		builders.TransactionReceipt().WithRandomHash().Build(),
	}

	outputs := []io.Writer{os.Stdout, ioutil.Discard}

	for _, output := range outputs {
		b.Run(reflect.TypeOf(output).String(), func(b *testing.B) {

			serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
				WithOutput(log.NewOutput(output).WithFormatter(log.NewHumanReadableFormatter()))

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				serviceLogger.Info("Benchmark test", log.StringableSlice("a-collection", receipts))
			}
			b.StopTimer()
		})
	}
}

func checkOutput(t *testing.T, output string) {
	jsonMap := parseOutput(output)

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestMultipleOutputs.func1", jsonMap["function"])
	require.Equal(t, "Service initialized", jsonMap["message"])
	require.NotEmpty(t, jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
}
