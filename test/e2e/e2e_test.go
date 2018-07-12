package e2e

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

type E2EConfig struct {
	Bootstrap   bool
	ApiEndpoint string
}

func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

func getConfig() E2EConfig {
	Bootstrap := len(os.Getenv("API_ENDPOINT")) == 0
	ApiEndpoint := "http://localhost:8080/api/"

	if !Bootstrap {
		ApiEndpoint = os.Getenv("API_ENDPOINT")
	}

	return E2EConfig{
		Bootstrap,
		ApiEndpoint,
	}
}

var _ = Describe("The Orbs Network", func() {
	It("accepts a transaction and reflects the state change after it is committed", func(done Done) {
		var node bootstrap.Node

		if getConfig().Bootstrap {
			gossipTransport := gossipAdapter.NewTamperingTransport()
			node = bootstrap.NewNode(":8080", "node1", gossipTransport, true, 1)
		}

		tx := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "transfer",
			InputArguments: []*protocol.MethodArgumentBuilder{
				{Name: "amount", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 17},
			},
		}

		_ = sendTransaction(tx)

		m := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "getBalance",
		}

		Eventually(func() uint64 {
			return callMethod(m).ClientResponse.OutputArgumentsIterator().NextOutputArguments().Uint64Value()
		}).Should(BeEquivalentTo(17))

		if getConfig().Bootstrap {
			node.GracefulShutdown(1 * time.Second)
		}

		close(done)
	}, 10)
})

func sendTransaction(txBuilder *protocol.TransactionBuilder) *services.SendTransactionOutput {
	input := (&client.SendTransactionRequestBuilder{
		SignedTransaction: &protocol.SignedTransactionBuilder{
			Transaction: txBuilder,
		}}).Build()

	return &services.SendTransactionOutput{ClientResponse: client.SendTransactionResponseReader(httpPost(input, "send-transaction"))}
}

func callMethod(txBuilder *protocol.TransactionBuilder) *services.CallMethodOutput {
	input := (&client.CallMethodRequestBuilder{
		Transaction: txBuilder,
	}).Build()

	return &services.CallMethodOutput{ClientResponse: client.CallMethodResponseReader(httpPost(input, "call-method"))}

}

func httpPost(input membuffers.Message, method string) []byte {
	res, err := http.Post(getConfig().ApiEndpoint+method, "application/octet-stream", bytes.NewReader(input.Raw()))
	Expect(err).ToNot(HaveOccurred())
	Expect(res.StatusCode).To(Equal(http.StatusOK))

	bytes, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	Expect(err).ToNot(HaveOccurred())

	return bytes
}
