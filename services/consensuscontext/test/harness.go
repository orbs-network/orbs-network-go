package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type harness struct {
	transactionPool *services.MockTransactionPool
	reporting       log.BasicLogger
	service         services.ConsensusContext
	config          consensuscontext.Config
}

func (h *harness) requestTransactionsBlock() (*protocol.TransactionsBlockContainer, error) {
	output, err := h.service.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{
		BlockHeight:             1,
		MaxBlockSizeKb:          0,
		MaxNumberOfTransactions: 0,
		PrevBlockHash:           hash.CalcSha256([]byte{1}),
	})
	if err != nil {
		return nil, err
	}
	return output.TransactionsBlock, nil
}

func (h *harness) expectTransactionsRequestedFromTransactionPool(numTransactionsToReturn int) {

	output := &services.GetTransactionsForOrderingOutput{
		SignedTransactions: nil,
	}

	for i := 0; i < numTransactionsToReturn; i++ {
		output.SignedTransactions = append(output.SignedTransactions, builders.TransferTransaction().WithAmount(uint64(i+1)*10).Build())
	}

	h.transactionPool.When("GetTransactionsForOrdering", mock.Any).Return(output, nil).Times(1)
}

func (h *harness) expectTransactionsNoLongerRequestedFromTransactionPool() {
	h.transactionPool.When("GetTransactionsForOrdering", mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyTransactionsRequestedFromTransactionPool(t *testing.T) {
	ok, _ := h.transactionPool.Verify()

	// TODO: How to print err if it's sometimes nil
	require.True(t, ok)
}

func newHarness() *harness {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	transactionPool := &services.MockTransactionPool{}

	serviceConfig := config.NewConsensusContextConfig(300, 2)

	service := consensuscontext.NewConsensusContext(transactionPool, nil, nil,
		serviceConfig, log)

	return &harness{
		transactionPool: transactionPool,
		reporting:       log,
		service:         service,
		config:          serviceConfig,
	}
}
