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
		httpServer := bootstrap.NewHttpServer(":8080", "node1", true)

		res, err := http.Post("http://localhost:8080/api/send_transaction?amount=17", "text/plain", nil)

		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))

		res, _ = http.Get("http://localhost:8080/api/call_method")

		Expect(ResponseBodyAsString(res)).To(Equal("17"))

		httpServer.Stop()

		close(done)
	}, 1)
})