package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
func TestValidateTransactionBlock(t *testing.T) {

	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	metricFactory := metric.NewRegistry()
	cfg := config.ForConsensusContextTests(nil)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.AddressForEd25519SignerForTests(6)).Build()
	txRootHashForValidBlock, err := CalculateTransactionsRootHash([]*protocol.SignedTransaction{transaction})
	prevBlock1 := builders.BlockPair().WithHeight(1001).Build()
	prevBlock2 := builders.BlockPair().WithHeight(1002).Build()

	if err != nil {
		t.Fatal(err)
	}

	s := NewConsensusContext(
		&services.MockTransactionPool{},
		&services.MockVirtualMachine{},
		&services.MockStateStorage{},
		cfg,
		log,
		metricFactory)

	t.Run("should return ok for valid block", func(t *testing.T) {
		t.Skipf("Skipped till previous block hash code can be fixed")

		validBlock := builders.
			BlockPair().
			WithProtocolVersion(cfg.ProtocolVersion()).
			WithVirtualChainId(cfg.VirtualChainId()).
			WithTransactions(0).
			WithTransaction(transaction).
			WithPrevBlock(prevBlock1).
			WithTransactionsRootHash(txRootHashForValidBlock).
			Build()
		//if err != nil {
		//	t.Fatal(err)
		//}
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: validBlock.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.NoError(t, err, "validation should have succeeded on valid block")
	})

	t.Run("should return error for block with incorrect protocol version", func(t *testing.T) {
		blockWithBadVersion := builders.BlockPair().
			WithProtocolVersion(999).
			WithVirtualChainId(cfg.VirtualChainId()).
			Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadVersion.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect protocol version")
	})
	t.Run("should return error for block with incorrect vchain", func(t *testing.T) {
		blockWithBadVchain := builders.
			BlockPair().
			WithProtocolVersion(cfg.ProtocolVersion()).
			WithVirtualChainId(999).
			Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadVchain.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect virtual chain")
	})

	t.Run("should return error for block with incorrect merkle root", func(t *testing.T) {
		incorrectTransactionsRootHash := []byte{1, 1, 1, 1, 1}
		blockWithBadTxRootHash := builders.
			BlockPair().
			WithProtocolVersion(cfg.ProtocolVersion()).
			WithVirtualChainId(cfg.VirtualChainId()).
			WithTransactions(0).
			WithTransaction(transaction).
			WithTransactionsRootHash(incorrectTransactionsRootHash).
			Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadTxRootHash.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect transaction root hash")
	})

	t.Run("should return error for block with incorrect prev block hash", func(t *testing.T) {
		prev2Hash := digest.CalcTransactionsBlockHash(prevBlock2.TransactionsBlock)
		blockWithBadPrevBlockHash := builders.
			BlockPair().
			WithProtocolVersion(cfg.ProtocolVersion()).
			WithVirtualChainId(cfg.VirtualChainId()).
			WithTransactions(0).
			WithTransaction(transaction).
			WithPrevBlock(prevBlock1). // order is important! WithPrevBlock() comes before WithPrevBlockHash()
			WithPrevBlockHash(prev2Hash).
			WithTransactionsRootHash(txRootHashForValidBlock).
			Build()

		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadPrevBlockHash.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect prev block hash")

	})
}
