package acceptance

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/gossip"
)

var _ = Describe("a leader node", func() {

	It("commits transactions to all nodes", func() {
		network := harness.CreateTestNetwork()

		network.SendTransaction(network.Leader(), &types.Transaction{Value: 17})
		network.SendTransaction(network.Leader(), &types.Transaction{Value: 97, Invalid: true})
		network.SendTransaction(network.Leader(), &types.Transaction{Value: 22})

		network.LeaderBp().WaitForBlocks(2)
		Expect(<- network.CallMethod(network.Leader())).To(Equal(39))

		network.ValidatorBp().WaitForBlocks(2)
		Expect(<- network.CallMethod(network.Validator())).To(Equal(39))

	})
})

var _ = Describe("a non-leader (validator) node", func() {

	It("propagates transactions to leader but does not commit them itself", func() {
		network := harness.CreateTestNetwork()

		network.Gossip().Pause(gossip.ForwardTransactionMessage)
		network.SendTransaction(network.Validator(), &types.Transaction{Value: 17})

		Expect(<- network.CallMethod(network.Leader())).To(Equal(0))
		Expect(<- network.CallMethod(network.Validator())).To(Equal(0))

		network.Gossip().Resume(gossip.ForwardTransactionMessage)
		network.LeaderBp().WaitForBlocks(1)
		Expect(<- network.CallMethod(network.Leader())).To(Equal(17))
		network.ValidatorBp().WaitForBlocks(1)
		Expect(<- network.CallMethod(network.Validator())).To(Equal(17))
	})

})
