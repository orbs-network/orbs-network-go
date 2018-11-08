package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var federationNodePublicKeysForTest = []primitives.Ed25519PublicKey{
	primitives.Ed25519PublicKey("dfc06c5be24a67adee80b35ab4f147bb1a35c55ff85eda69f40ef827bddec173"),
	primitives.Ed25519PublicKey("92d469d7c004cc0b24a192d9457836bf38effa27536627ef60718b00b0f33152"),
	primitives.Ed25519PublicKey("a899b318e65915aa2de02841eeb72fe51fddad96014b73800ca788a547f8cce0"),
	primitives.Ed25519PublicKey("58e7ed8169a151602b1349c990c84ca2fb2f62eb17378f9a94e49552fbafb9d8"),
	primitives.Ed25519PublicKey("23f97918acf48728d3f25a39a5f091a1a9574c52ccb20b9bad81306bd2af4631"),
	primitives.Ed25519PublicKey("07492c6612f78a47d7b6a18a17792a01917dec7497bdac1a35c477fbccc3303b"),
	primitives.Ed25519PublicKey("43a4dbbf7a672c6689dbdd662fd89a675214b00d884bb7113d3410b502ecd826"),
	primitives.Ed25519PublicKey("469bd276271aa6d59e387018cf76bd00f55c702931c13e80896eec8a32b22082"),
	primitives.Ed25519PublicKey("102073b28749be1e3daf5e5947605ec7d43c3183edb48a3aac4c9542cdbaf748"),
	primitives.Ed25519PublicKey("70d92324eb8d24b7c7ed646e1996f94dcd52934a031935b9ac2d0e5bbcfa357c"),
}

type harness struct {
	transactionPool *services.MockTransactionPool
	reporting       log.BasicLogger
	service         services.ConsensusContext
	config          config.ConsensusContextConfig
}

func (h *harness) requestTransactionsBlock(ctx context.Context) (*protocol.TransactionsBlockContainer, error) {
	output, err := h.service.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
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

func (h *harness) expectTransactionsRequestedFromTransactionPool(numTransactionsToReturn uint32) {

	output := &services.GetTransactionsForOrderingOutput{
		SignedTransactions: nil,
	}

	for i := uint32(0); i < numTransactionsToReturn; i++ {
		targetAddress := builders.AddressForEd25519SignerForTests(2)
		output.SignedTransactions = append(output.SignedTransactions, builders.TransferTransaction().WithAmountAndTargetAddress(uint64(i+1)*10, targetAddress).Build())
	}

	h.transactionPool.When("GetTransactionsForOrdering", mock.Any, mock.Any).Return(output, nil).Times(1)
}

func (h *harness) expectTransactionsNoLongerRequestedFromTransactionPool() {
	h.transactionPool.When("GetTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil).Times(0)
}

func (h *harness) verifyTransactionsRequestedFromTransactionPool(t *testing.T) {
	ok, _ := h.transactionPool.Verify()

	// TODO: How to print err if it's sometimes nil
	require.True(t, ok)
}

func newHarness() *harness {
	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))

	transactionPool := &services.MockTransactionPool{}
	federationNodes := make(map[string]config.FederationNode)
	for _, pk := range federationNodePublicKeysForTest {
		federationNodes[pk.KeyForMap()] = config.NewHardCodedFederationNode(pk)
	}

	cfg := config.ForConsensusContextTests(federationNodes)

	metricFactory := metric.NewRegistry()

	service := consensuscontext.NewConsensusContext(transactionPool, nil, nil,
		cfg, log, metricFactory)

	return &harness{
		transactionPool: transactionPool,
		reporting:       log,
		service:         service,
		config:          cfg,
	}
}
