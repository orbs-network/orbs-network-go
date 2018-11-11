package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/builders"
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
	s := NewConsensusContext(
		&services.MockTransactionPool{},
		&services.MockVirtualMachine{},
		&services.MockStateStorage{},
		cfg,
		log,
		metricFactory)

	t.Run("should return ok for valid block", func(t *testing.T) {
		validBlock := builders.BlockPair().Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: validBlock.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.NoError(t, err, "validation should have succeeded on valid block")
	})

	t.Run("should return error for block with incorrect protocol version", func(t *testing.T) {
		blockWithBadVersion := builders.BlockPair().WithProtocolVersion(999).Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadVersion.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect protocol version")
	})
	t.Run("should return error for block with incorrect vchain", func(t *testing.T) {
		blockWithBadVersion := builders.BlockPair().WithVirtualChainId(999).Build()
		input := &services.ValidateTransactionsBlockInput{
			TransactionsBlock: blockWithBadVersion.TransactionsBlock,
			PrevBlockHash:     nil,
		}
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should have failed on incorrect virtual chain")
	})

}
