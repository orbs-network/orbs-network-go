package gamma

import (
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"regexp"
)

type harness struct {
	gamma *gamma.GammaServer
	port  int
}

func (h *harness) shutdown() {
	h.gamma.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
}

func newHarness() *harness {
	server := gamma.StartGammaServer(":0", false)
	return &harness{gamma: server, port: server.Port()}
}

func extractTxIdFromSendTxOutput(out string) string {
	re := regexp.MustCompile(`\"TxId\":\s+\"(\w+)\"`)
	res := re.FindStringSubmatch(out)
	return res[1]
}
