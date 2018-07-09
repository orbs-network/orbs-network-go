package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"time"
	"io/ioutil"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"bytes"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"net/http"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = Describe("The Orbs Network", func() {
	It("accepts a transaction and reflects the state change after it is committed", func(done Done) {
		node := bootstrap.NewNode(":8080", "node1", true, 1)

		tx := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "transfer",
			InputArguments: []*protocol.MethodArgumentBuilder{
				{Name: "amount", Type: protocol.MethodArgumentTypeUint64, Uint64: 17},
			},
		}

		_ = sendTransaction(tx)

		m := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "getBalance",
		}

		Eventually(func() uint64 {
			return callMethod(m).ClientResponse.OutputArgumentsIterator().NextOutputArguments().Uint64()
		}).Should(BeEquivalentTo(17))

		node.GracefulShutdown(1 * time.Second)

		close(done)
	}, 10)
})

func sendTransaction(txBuilder *protocol.TransactionBuilder) *services.SendTransactionOutput {
	input := (&client.SendTransactionRequestBuilder{
		SignedTransaction: &protocol.SignedTransactionBuilder{
			Transaction: txBuilder,
		}}).Build()

	return &services.SendTransactionOutput{ClientResponse: client.SendTransactionResponseReader(httpPost(input,"send-transaction"))}
}

func callMethod(txBuilder *protocol.TransactionBuilder) *services.CallMethodOutput {
	input := (&client.CallMethodRequestBuilder{
		Transaction: txBuilder,
	}).Build()

	return &services.CallMethodOutput{ClientResponse: client.CallMethodResponseReader(httpPost(input,"call-method"))}

}

func httpPost(input membuffers.Message, method string) []byte {
	res, err := http.Post("http://localhost:8080/api/" + method, "application/octet-stream", bytes.NewReader(input.Raw()))
	Expect(err).ToNot(HaveOccurred())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	Expect(err).ToNot(HaveOccurred())

	return bytes
}

