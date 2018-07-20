package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

var _ = Describe("a leader node", func() {

	It("must get validations by all nodes to commit a transaction", func(done Done) {
		// leader is nodeIndex 0, validator is nodeIndex 1
		network := harness.NewTestNetwork(2)
		defer network.FlushLog()

		prePrepareLatch := network.GossipTransport().LatchOn(adapter.ConsensusMessage(gossipmessages.LEAN_HELIX_PRE_PREPARE))
		prePrepareTamper := network.GossipTransport().Fail(adapter.ConsensusMessage(gossipmessages.LEAN_HELIX_PRE_PREPARE))
		<-network.SendTransfer(0, 17)

		prePrepareLatch.Wait()
		Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(0))
		Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(0))

		prePrepareTamper.Release()
		prePrepareLatch.Remove()

		network.BlockPersistence(0).WaitForBlocks(1)
		Expect(<-network.CallGetBalance(0)).To(BeEquivalentTo(17))

		network.BlockPersistence(1).WaitForBlocks(1)
		Expect(<-network.CallGetBalance(1)).To(BeEquivalentTo(17))

		close(done)
	}, 1)

})
