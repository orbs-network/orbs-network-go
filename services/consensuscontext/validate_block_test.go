package consensuscontext

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func withTxBlock(block *protocol.BlockPairContainer) *services.ValidateTransactionsBlockInput {
	input := &services.ValidateTransactionsBlockInput{
		TransactionsBlock: block.TransactionsBlock,
		PrevBlockHash:     block.TransactionsBlock.Header.PrevBlockHashPtr(),
		BlockHeight:       block.TransactionsBlock.Header.BlockHeight(),
	}
	return input
}

func inputs(cfg config.ConsensusContextConfig) (*protocol.BlockPairContainer, *services.ValidateTransactionsBlockInput) {

	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.AddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := calculateTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)
	//prevBlock2 := builders.BlockPair().WithHeight(1002).Build()
	//prev2Hash := digest.CalcTransactionsBlockHash(prevBlock2.TransactionsBlock)

	// include only one transaction in block
	block := builders.
		BlockPair().
		WithHeight(currentBlockHeight).
		WithProtocolVersion(cfg.ProtocolVersion()).
		WithVirtualChainId(cfg.VirtualChainId()).
		WithTransactions(0).
		WithTransaction(transaction).
		WithPrevBlock(validPrevBlock).
		WithPrevBlockHash(validPrevBlockHash).
		WithMetadata(txMetadata).
		WithMetadataHash(validMetadataHash).
		WithTransactionsRootHash(txRootHashForValidBlock).
		Build()

	input := &services.ValidateTransactionsBlockInput{
		BlockHeight:        currentBlockHeight,
		TransactionsBlock:  nil, // fill in each test
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return block, input
}

// func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
func TestValidateTransactionBlock(t *testing.T) {

	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	metricFactory := metric.NewRegistry()
	cfg := config.ForConsensusContextTests(nil)
	txPool := &services.MockTransactionPool{}
	txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)

	empty32ByteHash := make([]byte, 32)

	falsyValidateTransactionOrdering := func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (
		*services.ValidateTransactionsForOrderingOutput, error) {
		return &services.ValidateTransactionsForOrderingOutput{}, errors.New("Some error")
	}

	s := NewConsensusContext(
		txPool,
		&services.MockVirtualMachine{},
		&services.MockStateStorage{},
		cfg,
		log,
		metricFactory)

	t.Run("should return ok for valid block", func(t *testing.T) {
		block, input := inputs(cfg)
		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.NoError(t, err, "validation should succeed on valid block")
	})

	t.Run("should return error for block with incorrect protocol version", func(t *testing.T) {
		block, input := inputs(cfg)
		if err := block.TransactionsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should fail on incorrect protocol version")
	})

	t.Run("should return error for block with incorrect virtual chain", func(t *testing.T) {
		block, input := inputs(cfg)
		if err := block.TransactionsBlock.Header.MutateVirtualChainId(999); err != nil {
			t.Error(err)
		}
		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should fail on incorrect virtual chain")
	})

	t.Run("should return error for block with incorrect merkle root", func(t *testing.T) {
		block, input := inputs(cfg)
		if err := block.TransactionsBlock.Header.MutateTransactionsRootHash(empty32ByteHash); err != nil {
			t.Error(err)
		}

		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should fail on incorrect transaction root hash")
	})

	t.Run("should return error for block with incorrect prev block hash", func(t *testing.T) {
		block, input := inputs(cfg)
		if err := block.TransactionsBlock.Header.MutatePrevBlockHashPtr(empty32ByteHash); err != nil {
			t.Error(err)
		}
		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should fail on incorrect prev block hash")
	})

	t.Run("should return error for block with incorrect metadata hash", func(t *testing.T) {
		block, input := inputs(cfg)
		if err := block.TransactionsBlock.Header.MutateMetadataHash(empty32ByteHash); err != nil {
			t.Error(err)
		}
		input.TransactionsBlock = block.TransactionsBlock
		_, err := s.ValidateTransactionsBlock(context.Background(), input)
		require.Error(t, err, "validation should fail on incorrect metadata hash")
	})

	t.Run("should return error for block with failing tx ordering validation", func(t *testing.T) {
		block, input := inputs(cfg)
		input.TransactionsBlock = block.TransactionsBlock
		err := ValidateTransactionsBlockInternal(
			context.Background(),
			input, cfg.ProtocolVersion(), cfg.VirtualChainId(), cfg.ConsensusContextSystemTimestampAllowedJitter(),
			falsyValidateTransactionOrdering)
		require.Error(t, err, "validation should fail on failing tx ordering validation")
	})
}

func TestIsValidBlockTimestamp(t *testing.T) {

	jitter := 2 * time.Second
	tests := []struct {
		name                        string
		currentBlockTimestampOffset time.Duration
		prevBlockTimestampOffset    time.Duration
		expectedToPass              bool
	}{
		{
			"Current block has valid timestamp",
			1 * time.Second,
			-3 * time.Second,
			true,
		},
		{
			"Current block is too far in the past",
			-3 * time.Second,
			-6 * time.Second,
			false,
		},
		{
			"Current block is too far in the future",
			3 * time.Second,
			-6 * time.Second,
			false,
		},
		{
			"Current block is older than prev block",
			-2 * time.Second,
			-1 * time.Second,
			false,
		},
		{
			"Current block is as old as prev block",
			-2 * time.Second,
			-2 * time.Second,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			currentBlockTimestamp := primitives.TimestampNano(now.Add(tt.currentBlockTimestampOffset).UnixNano())
			prevBlockTimestamp := primitives.TimestampNano(now.Add(tt.prevBlockTimestampOffset).UnixNano())
			res := isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, now, jitter)
			if tt.expectedToPass {
				require.True(t, res, tt.name)
			} else {
				require.False(t, res, tt.name)
			}
		})
	}
}
