package instrumentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"io"
	"os"
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

func parseStdout(input string) map[string]interface{} {
	jsonMap := make(map[string]interface{})
	_ = json.Unmarshal([]byte(input), &jsonMap)
	return jsonMap
}

func TestSimpleLogger(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer)
		serviceLogger.Info("Service initialized")
	})

	fmt.Println(stdout)
	jsonMap := parseStdout(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestSimpleLogger.func1"))
	Expect(jsonMap["message"]).To(Equal("Service initialized"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
}

func TestNestedLogger(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer)
		txId := String("txId", "1234567")
		txFlowLogger := serviceLogger.For(String("flow", TransactionFlow))
		txFlowLogger.Info(TransactionAccepted, txId, Bytes("payload", []byte{1, 2, 3, 99}))
	})

	fmt.Println(stdout)
	jsonMap := parseStdout(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestNestedLogger.func1"))
	Expect(jsonMap["message"]).To(Equal(TransactionAccepted))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["payload"]).To(Equal("MlZmV0E="))
}

func TestMeter(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer)
		txId := String("txId", "1234567")
		txFlowLogger := serviceLogger.For(String("flow", TransactionFlow))
		meter := txFlowLogger.Meter("tx-process-time", txId)
		defer meter.Done()

		time.Sleep(1 * time.Millisecond)
	})

	fmt.Println(stdout)

	jsonMap := parseStdout(stdout)

	Expect(jsonMap["level"]).To(Equal("metric"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestMeter.func1"))
	Expect(jsonMap["message"]).To(Equal("Metric recorded"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["metric"]).To(Equal("public-api-TransactionFlow-tx-process-time"))
	Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["process-time"]).NotTo(BeNil())
}

func TestCustomLogFormatter(t *testing.T) {
	RegisterTestingT(t)

	stdout := captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer).WithFormatter(NewHumanReadableFormatter())
		serviceLogger.Info("Service initialized", Int("some-int-value", 12), BlockHeight(primitives.BlockHeight(9999)), Bytes("bytes", []byte{2, 3, 99}), Stringable("vchainId", primitives.VirtualChainId(123)))
	})

	fmt.Println(stdout)

	Expect(stdout).To(HavePrefix("info"))
	Expect(stdout).To(ContainSubstring("Service initialized"))
	Expect(stdout).To(ContainSubstring("node=node1"))
	Expect(stdout).To(ContainSubstring("service=public-api"))
	Expect(stdout).To(ContainSubstring("blockHeight=270f"))
	Expect(stdout).To(ContainSubstring("vchainId=7b"))
	Expect(stdout).To(ContainSubstring("bytes=gDp"))
	Expect(stdout).To(ContainSubstring("some-int-value=12"))
	Expect(stdout).To(ContainSubstring("function=github.com/orbs-network/orbs-network-go/instrumentation.TestCustomLogFormatter.func1"))
	Expect(stdout).To(ContainSubstring("source="))
	Expect(stdout).To(ContainSubstring("orbs-network/orbs-network-go/instrumentation/basic_logger_test.go"))
}
