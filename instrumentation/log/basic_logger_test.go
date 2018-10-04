package log_test

import (
	"bytes"
	"encoding/json"
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
)

const (
	TransactionFlow     = "TransactionFlow"
	TransactionAccepted = "Transaction accepted"
)

func parseOutput(input string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(input), &jsonMap)
	return jsonMap
}

func TestSimpleLogger(t *testing.T) {
	b := new(bytes.Buffer)
	log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b)).Info("Service initialized")

	jsonMap := parseOutput(b.String())

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestSimpleLogger", jsonMap["function"])
	require.Equal(t, "Service initialized", jsonMap["message"])
	require.Regexp(t, "^instrumentation/log/basic_logger_test.go", jsonMap["source"])
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
	b := new(bytes.Buffer)
	log.GetLogger().WithOutput(log.NewOutput(b)).
		LogFailedExpectation("Service initialized compare", log.BlockHeight(primitives.BlockHeight(9999)), log.BlockHeight(primitives.BlockHeight(8888)), log.Bytes("bytes", []byte{2, 3, 99}))

	jsonMap := parseOutput(b.String())

	require.Equal(t, "expectation", jsonMap["level"])
	require.Equal(t, "020363", jsonMap["bytes"])
	require.Equal(t, "22b8", jsonMap["actual-block-height"])
	require.Equal(t, "270f", jsonMap["expected-block-height"])
}

func TestNestedLogger(t *testing.T) {
	b := new(bytes.Buffer)

	txId := log.String("txId", "1234567")
	txFlowLogger := log.GetLogger().WithOutput(log.NewOutput(b)).WithTags(log.String("flow", TransactionFlow))
	txFlowLogger.Info(TransactionAccepted, txId, log.Bytes("payload", []byte{1, 2, 3, 99, 250}))

	jsonMap := parseOutput(b.String())

	require.Equal(t, TransactionAccepted, jsonMap["message"])
	require.Equal(t, "1234567", jsonMap["txId"])
	require.Equal(t, TransactionFlow, jsonMap["flow"])
	require.Equal(t, "01020363fa", jsonMap["payload"])

}

func TestStringableSlice(t *testing.T) {
	b := new(bytes.Buffer)
	var receipts = []*protocol.TransactionReceipt{builders.TransactionReceipt().Build(), builders.TransactionReceipt().Build()}

	log.GetLogger().WithOutput(log.NewOutput(b)).Info("StringableSlice test", log.StringableSlice("a-collection", receipts))

	jsonMap := parseOutput(b.String())

	require.Equal(t, []interface{}{
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,}",
		"{Txhash:736f6d652d74782d68617368,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,}",
	}, jsonMap["a-collection"])
}

func TestCustomLogFormatter(t *testing.T) {
	b := new(bytes.Buffer)
	serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b).WithFormatter(log.NewHumanReadableFormatter()))
	serviceLogger.Info("Service initialized",
		log.Int("some-int-value", 12),
		log.BlockHeight(primitives.BlockHeight(9999)),
		log.Bytes("bytes", []byte{2, 3, 99}),
		log.Stringable("vchainId", primitives.VirtualChainId(123)),
		log.String("_test-id", "hello"),
		log.String("_underscore", "wow"))

	out := b.String()

	require.Regexp(t, "^info", out)
	require.Regexp(t, "Service initialized", out)
	require.Regexp(t, "node=node1", out)
	require.Regexp(t, "service=public-api", out)
	require.Regexp(t, "block-height=270f", out)
	require.Regexp(t, "vchainId=7b", out)
	require.Regexp(t, "bytes=gDp", out)
	require.Regexp(t, "some-int-value=12", out)
	require.Regexp(t, "function=log_test.TestCustomLogFormatter", out)
	require.Regexp(t, "source=instrumentation/log/basic_logger_test.go", out)
	require.Regexp(t, "_test-id=hello", out)
	require.Regexp(t, "_underscore=wow", out)
}

func TestHumanReadableFormatterFormatWithStringableSlice(t *testing.T) {
	b := new(bytes.Buffer)
	transactions := []*protocol.SignedTransaction{builders.TransferTransaction().Build()}

	log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b).WithFormatter(log.NewHumanReadableFormatter())).
		Info("StringableSlice HR test", log.StringableSlice("a-collection", transactions))

	out := b.String()

	require.Regexp(t, "a-collection=", out)
	require.Regexp(t, "{Transaction:{ProtocolVersion:1,", out)
}

func TestMeter(t *testing.T) {
	b := new(bytes.Buffer)

	serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b))
	txId := log.String("txId", "1234567")
	txFlowLogger := serviceLogger.WithTags(log.String("flow", TransactionFlow))
	meter := txFlowLogger.Meter("tx-process-time", txId)
	meter.Done()

	jsonMap := parseOutput(b.String())

	require.Equal(t, "metric", jsonMap["level"])
	require.Equal(t, "Metric recorded", jsonMap["message"])
	require.Equal(t, "public-api-TransactionFlow-tx-process-time", jsonMap["metric"])
	require.Equal(t, "1234567", jsonMap["txId"])
	require.Equal(t, TransactionFlow, jsonMap["flow"])
	require.NotNil(t, jsonMap["process-time"])
}

func TestMultipleOutputs(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "logger_test_multiple_outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name()) // clean up

	fileOutput, _ := os.Create(tempFile.Name())

	b := new(bytes.Buffer)

	log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b), log.NewOutput(fileOutput)).
		Info("Service initialized")

	rawFile, _ := ioutil.ReadFile(tempFile.Name())
	fileContents := string(rawFile)

	checkOutput := func(output string) {
		jsonMap := parseOutput(output)

		require.Equal(t, "info", jsonMap["level"])
		require.Equal(t, "node1", jsonMap["node"])
		require.Equal(t, "public-api", jsonMap["service"])
		require.Equal(t, "log_test.TestMultipleOutputs", jsonMap["function"])
		require.Equal(t, "Service initialized", jsonMap["message"])
		require.NotEmpty(t, jsonMap["source"])
		require.NotNil(t, jsonMap["timestamp"])
	}

	checkOutput(b.String())
	checkOutput(fileContents)
}

func TestMultipleOutputsForMemoryViolationByHumanReadable(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "logger_test_multiple_outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name()) // clean up

	b := new(bytes.Buffer)

	fileOutput, _ := os.Create(tempFile.Name())

	require.NotPanics(t, func() {
		log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewOutput(b).WithFormatter(log.NewHumanReadableFormatter()), log.NewOutput(fileOutput)).
			Info("Service initialized")
	})
}

func BenchmarkBasicLoggerInfoFormatters(b *testing.B) {
	receipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().WithRandomHash().Build(),
		builders.TransactionReceipt().WithRandomHash().Build(),
	}

	formatters := []log.LogFormatter{log.NewHumanReadableFormatter(), log.NewJsonFormatter()}

	for _, formatter := range formatters {
		b.Run(reflect.TypeOf(formatter).String(), func(b *testing.B) {
			serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
				WithOutput(log.NewOutput(ioutil.Discard).WithFormatter(formatter))

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				serviceLogger.Info("Benchmark test", log.StringableSlice("a-collection", receipts))
			}
			b.StopTimer()
		})
	}
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
