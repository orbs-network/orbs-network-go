package e2e

import (
. "github.com/onsi/ginkgo"
. "github.com/onsi/gomega"
"testing"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"net/http"
	"github.com/onsi/gomega/gbytes"
	"time"
	"io/ioutil"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"bytes"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

func ResponseBodyAsString(resp *http.Response) string {
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(gbytes.TimeoutReader(resp.Body, 10 * time.Second))
	if err == gbytes.ErrTimeout {
		return "<timeout>"
	}
	if err != nil {
		return "<error>"
	}
	return string(bodyBytes)
}

var _ = Describe("The Orbs Network", func() {
	It("accepts a transaction and reflects the state change after it is committed", func(done Done) {
		node := bootstrap.NewNode(":8080", "node1", true, 1)

		tx := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "transfer",
			InputArgument: []*protocol.MethodArgumentBuilder{
				{Name: "amount", Type: protocol.MethodArgumentTypeUint64, Uint64: 17},
			},
		}

		_ = sendTransaction(tx)

		m := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "getBalance",
		}

		Eventually(func() uint64 {
			return callMethod(m).OutputArgumentIterator().NextOutputArgument().TypeUint64()
		}).Should(Equal(17))

		node.GracefulShutdown(1 * time.Second)

		close(done)
	}, 10)
})

func sendTransaction(txBuilder *protocol.TransactionBuilder) *services.SendTransactionOutput {
	reader := bytes.NewReader((&services.SendTransactionInputBuilder{SignedTransaction: &protocol.SignedTransactionBuilder{TransactionContent: txBuilder}}).Build().Raw())

	res, err := http.Post("http://localhost:8080/api/send-transaction", "application/octet-stream", reader)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	Expect(err).ToNot(HaveOccurred())

	return services.SendTransactionOutputReader(bytes)
}

func callMethod(txBuilder *protocol.TransactionBuilder) *services.CallMethodOutput {
	reader := bytes.NewReader((&services.CallMethodInputBuilder{Transaction: txBuilder}).Build().Raw())

	res, err := http.Post("http://localhost:8080/api/call-method", "application/octet-stream", reader)
	Expect(err).ToNot(HaveOccurred())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	Expect(err).ToNot(HaveOccurred())

	return services.CallMethodOutputReader(bytes)}