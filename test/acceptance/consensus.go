package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		// leader is nodeIndex 0, validator is nodeIndex 1
		network := harness.NewTestNetwork(2)
		defer network.FlushLog()
		consensusRound := network.LoopControl(0).LatchFor("consensus_round")

		consensusRound.Brake()
		network.GossipTransport().Fail(gossipmessages.HEADER_TOPIC_LEAN_HELIX, uint16(gossipmessages.LEAN_HELIX_PRE_PREPARE))
		<-network.SendTransfer(0, 17)

		consensusRound.Tick()
		Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(0))
		Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(0))

		network.GossipTransport().Pass(gossipmessages.HEADER_TOPIC_LEAN_HELIX, uint16(gossipmessages.LEAN_HELIX_PRE_PREPARE))
		consensusRound.Release()

		network.BlockPersistence(0).WaitForBlocks(1)
		Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(17))

		network.BlockPersistence(1).WaitForBlocks(1)
		Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(17))
		close(done)
	}, 1)

})
