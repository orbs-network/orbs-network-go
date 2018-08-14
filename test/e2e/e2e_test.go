package e2e

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
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

		// TODO: kill me - why do we need this override?
		if getConfig().Bootstrap {
			gossipTransport := gossipAdapter.NewTamperingTransport()
			nodeKeyPair := keys.Ed25519KeyPairForTests(0)
			node = bootstrap.NewNode(
				":8080",
				nodeKeyPair.PublicKey(),
				nodeKeyPair.PrivateKey(),
				map[string]config.FederationNode{nodeKeyPair.PublicKey().KeyForMap(): config.NewHardCodedFederationNode(nodeKeyPair.PublicKey())},
				70,
				nodeKeyPair.PublicKey(), // we are the leader
				consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
				2*1000,
				gossipTransport,
				5,
				3,
				300,
				300,
				1,
			)

			// To let node start up properly, otherwise in Docker we get connection refused
			time.Sleep(100 * time.Millisecond)
		}

		tx := builders.TransferTransaction().WithAmount(17).Builder()

		_ = sendTransaction(tx)

		m := &protocol.TransactionBuilder{
			ContractName: "BenchmarkToken",
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

func sendTransaction(txBuilder *protocol.SignedTransactionBuilder) *services.SendTransactionOutput {
	input := (&client.SendTransactionRequestBuilder{
		SignedTransaction: txBuilder,
	}).Build()

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
