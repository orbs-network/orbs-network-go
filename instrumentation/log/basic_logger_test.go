// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
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

func parseOutput(input string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(input), &jsonMap)
	return jsonMap
}

func TestBasicLogger_WithTags_ClonesLoggerFully(t *testing.T) {
	v1 := log.String("k1", "v1")
	v2 := log.String("c1", "v2")
	v3 := log.String("c2", "v3")

	parent := log.GetLogger(v1)
	child1 := parent.WithTags(v2)
	child2 := parent.WithTags(v3)

	require.ElementsMatch(t, []*log.Field{v1}, parent.Tags())
	require.ElementsMatch(t, []*log.Field{v1, v2}, child1.Tags())
	require.ElementsMatch(t, []*log.Field{v1, v3}, child2.Tags())
}

func TestSimpleLogger(t *testing.T) {
	b := new(bytes.Buffer)
	log.GetLogger(log.Node("node1"), log.VirtualChainId(primitives.VirtualChainId(999)), log.Service("public-api")).WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).Info("Service initialized")

	jsonMap := parseOutput(b.String())

	require.Equal(t, "info", jsonMap["level"])
	require.Equal(t, "node1", jsonMap["node"])
	require.Equal(t, 999.0, jsonMap["vcid"]) // because golang JSON parser decodes ints as float64
	require.Equal(t, "public-api", jsonMap["service"])
	require.Equal(t, "log_test.TestSimpleLogger", jsonMap["function"])
	require.Equal(t, "Service initialized", jsonMap["message"])
	require.Regexp(t, "^instrumentation/log/basic_logger_test.go", jsonMap["source"])
	require.NotNil(t, jsonMap["timestamp"])
}

func TestSimpleLogger_AggregateField(t *testing.T) {
	ctx := trace.NewContext(context.Background(), "foo")
	b := new(bytes.Buffer)
	log.GetLogger().
		WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).
		Info("bar", trace.LogFieldFrom(ctx))

	jsonMap := parseOutput(b.String())

	require.Equal(t, "foo", jsonMap["entry-point"])
	require.NotEmpty(t, jsonMap[trace.RequestId])

}

func TestSimpleLogger_AggregateField_NestedLogger(t *testing.T) {
	ctx := trace.NewContext(context.Background(), "foo")
	b := new(bytes.Buffer)
	log.GetLogger(log.String("k1", "v1")).
		WithTags(trace.LogFieldFrom(ctx)).
		WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).
		Info("bar")

	jsonMap := parseOutput(b.String())

	require.Equal(t, "foo", jsonMap["entry-point"])
	require.Equal(t, "v1", jsonMap["k1"])
	require.NotEmpty(t, jsonMap[trace.RequestId])

}

func TestBasicLogger_WithFilter(t *testing.T) {
	b := new(bytes.Buffer)
	log.GetLogger().WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).
		WithFilters(log.OnlyErrors()).
		Info("foo")
	require.Empty(t, b.String(), "output was not empty")
}

func TestCompareLogger(t *testing.T) {
	b := new(bytes.Buffer)
	log.GetLogger().WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).
		LogFailedExpectation("Service initialized compare", log.BlockHeight(primitives.BlockHeight(9999)), log.BlockHeight(primitives.BlockHeight(8888)), log.Bytes("bytes", []byte{2, 3, 99}))

	jsonMap := parseOutput(b.String())

	require.Equal(t, "expectation", jsonMap["level"])
	require.Equal(t, "020363", jsonMap["bytes"])
	require.Equal(t, float64(8888), jsonMap["actual-block-height"])
	require.Equal(t, float64(9999), jsonMap["expected-block-height"])
}

func TestNestedLogger(t *testing.T) {
	b := new(bytes.Buffer)

	txId := log.String("txId", "1234567")
	txFlowLogger := log.GetLogger().WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).WithTags(log.String("flow", TransactionFlow))
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

	log.GetLogger().WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter())).Info("StringableSlice test", log.StringableSlice("a-collection", receipts))

	jsonMap := parseOutput(b.String())

	require.Equal(t, []interface{}{
		"{Txhash:ab2eccdf91e87771d6a8a5a37a6d26a9a220f78b3aa0662842b682a869e0819a,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,OutputEventsArray:,}",
		"{Txhash:ab2eccdf91e87771d6a8a5a37a6d26a9a220f78b3aa0662842b682a869e0819a,ExecutionResult:EXECUTION_RESULT_SUCCESS,OutputArgumentArray:,OutputEventsArray:,}",
	}, jsonMap["a-collection"])
}

func TestCustomLogFormatter(t *testing.T) {
	b := new(bytes.Buffer)
	serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
		WithOutput(log.NewFormattingOutput(b, log.NewHumanReadableFormatter()))
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
	require.Regexp(t, "block-height=9999", out)
	require.Regexp(t, "vchainId=7b", out)
	require.Regexp(t, "bytes=020363", out)
	require.Regexp(t, "some-int-value=12", out)
	require.Regexp(t, "function=log_test.TestCustomLogFormatter", out)
	require.Regexp(t, "source=instrumentation/log/basic_logger_test.go", out)
	require.Regexp(t, "_test-id=hello", out)
	require.Regexp(t, "_underscore=wow", out)
}

func TestHumanReadable_AggregateField(t *testing.T) {
	ctx := trace.NewContext(context.Background(), "foo")
	b := new(bytes.Buffer)
	log.GetLogger().
		WithOutput(log.NewFormattingOutput(b, log.NewHumanReadableFormatter())).
		Info("bar", trace.LogFieldFrom(ctx))

	out := b.String()
	require.Regexp(t, "entry-point=foo", out)
	require.Regexp(t, trace.RequestId+"=foo.*", out)

}

func TestHumanReadableFormatterFormatWithStringableSlice(t *testing.T) {
	b := new(bytes.Buffer)
	transactions := []*protocol.SignedTransaction{builders.TransferTransaction().Build()}

	log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewFormattingOutput(b, log.NewHumanReadableFormatter())).
		Info("StringableSlice HR test", log.StringableSlice("a-collection", transactions))

	out := b.String()

	require.Regexp(t, "a-collection=", out)
	require.Regexp(t, "{Transaction:{ProtocolVersion:1,", out)
}

func TestMultipleOutputs(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "logger_test_multiple_outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name()) // clean up

	fileOutput, _ := os.Create(tempFile.Name())

	b := new(bytes.Buffer)

	log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewFormattingOutput(b, log.NewJsonFormatter()), log.NewFormattingOutput(fileOutput, log.NewJsonFormatter())).
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
		log.GetLogger(log.Node("node1"), log.Service("public-api")).WithOutput(log.NewFormattingOutput(b, log.NewHumanReadableFormatter()), log.NewFormattingOutput(fileOutput, log.NewJsonFormatter())).
			Info("Service initialized")
	})
}

func TestJsonFormatterWithCustomTimestampColumn(t *testing.T) {
	f := log.NewJsonFormatter().WithTimestampColumn("@timestamp")
	row := f.FormatRow(time.Now(), "info", "hello")

	require.Regexp(t, "@timestamp", row)
}

func BenchmarkBasicLoggerInfoFormatters(b *testing.B) {
	ctrlRand := rand.NewControlledRand(b)

	receipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().WithRandomHash(ctrlRand).Build(),
		builders.TransactionReceipt().WithRandomHash(ctrlRand).Build(),
	}

	formatters := []log.LogFormatter{log.NewHumanReadableFormatter(), log.NewJsonFormatter()}

	for _, formatter := range formatters {
		b.Run(reflect.TypeOf(formatter).String(), func(b *testing.B) {
			serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
				WithOutput(log.NewFormattingOutput(ioutil.Discard, log.NewJsonFormatter()))

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				serviceLogger.Info("Benchmark test", log.StringableSlice("a-collection", receipts))
			}
			b.StopTimer()
		})
	}

}

func BenchmarkBasicLoggerInfoWithDevNull(b *testing.B) {
	ctrlRand := rand.NewControlledRand(b)

	receipts := []*protocol.TransactionReceipt{
		builders.TransactionReceipt().WithRandomHash(ctrlRand).Build(),
		builders.TransactionReceipt().WithRandomHash(ctrlRand).Build(),
	}

	outputs := []io.Writer{os.Stdout, ioutil.Discard}

	for _, output := range outputs {
		b.Run(reflect.TypeOf(output).String(), func(b *testing.B) {

			serviceLogger := log.GetLogger(log.Node("node1"), log.Service("public-api")).
				WithOutput(log.NewFormattingOutput(output, log.NewHumanReadableFormatter()))

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				serviceLogger.Info("Benchmark test", log.StringableSlice("a-collection", receipts))
			}
			b.StopTimer()
		})
	}
}
