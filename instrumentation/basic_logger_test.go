package instrumentation

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/onsi/gomega"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const (
	TransactionFlow     = "TransactionFlow"
	TransactionAccepted = "Transaction accepted"
)

func readString(reader io.Reader) string {
	result, err := ioutil.ReadAll(reader)
	fmt.Println(err)
	return string(result)
}

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

func TestReport(t *testing.T) {
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
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestReport.func1"))
	Expect(jsonMap["message"]).To(Equal("Service initialized"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())

	stdout = captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer)
		txId := String("txId", "1234567")
		txFlowLogger := serviceLogger.For(String("flow", TransactionFlow))
		txFlowLogger.Info(TransactionAccepted, txId, Bytes("payload", []byte{1, 2, 3, 99}))
	})

	fmt.Println(stdout)
	jsonMap = parseStdout(stdout)

	Expect(jsonMap["level"]).To(Equal("info"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestReport.func2"))
	Expect(jsonMap["message"]).To(Equal(TransactionAccepted))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["payload"]).To(Equal("MlZmV0E="))

	stdout = captureStdout(func(writer io.Writer) {
		serviceLogger := GetLogger(Node("node1"), Service("public-api")).WithOutput(writer)
		txId := String("txId", "1234567")
		txFlowLogger := serviceLogger.For(String("flow", TransactionFlow))
		meter := txFlowLogger.Meter("tx-process-time", txId)
		defer meter.Done()

		time.Sleep(1 * time.Millisecond)
	})

	fmt.Println(stdout)

	jsonMap = parseStdout(stdout)

	Expect(jsonMap["level"]).To(Equal("metric"))
	Expect(jsonMap["node"]).To(Equal("node1"))
	Expect(jsonMap["service"]).To(Equal("public-api"))
	Expect(jsonMap["function"]).To(Equal("github.com/orbs-network/orbs-network-go/instrumentation.TestReport.func3"))
	Expect(jsonMap["message"]).To(Equal("Metric recorded"))
	Expect(jsonMap["source"]).NotTo(BeEmpty())
	Expect(jsonMap["timestamp"]).NotTo(BeNil())
	Expect(jsonMap["metric"]).To(Equal("public-api-TransactionFlow-tx-process-time"))
	//FIXME pass txId as well
	//Expect(jsonMap["txId"]).To(Equal("1234567"))
	Expect(jsonMap["flow"]).To(Equal(TransactionFlow))
	Expect(jsonMap["process-time"]).NotTo(BeNil())
}
