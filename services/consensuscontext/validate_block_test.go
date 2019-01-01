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

func inputs(cfg config.ConsensusContextConfig) (*protocol.BlockPairContainer, *services.ValidateTransactionsBlockInput) {

	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := calculateTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)

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
		CurrentBlockHeight: currentBlockHeight,
		TransactionsBlock:  nil, // fill in each test
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return block, input
}

func toValidatorContext(cfg config.ConsensusContextConfig) *validatorContext {
	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := calculateTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)

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
		CurrentBlockHeight: currentBlockHeight,
		TransactionsBlock:  block.TransactionsBlock, // fill in each test
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return &validatorContext{
		protocolVersion:        cfg.ProtocolVersion(),
		virtualChainId:         cfg.VirtualChainId(),
		allowedTimestampJitter: cfg.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
	}
}

func TestTransactionBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	empty32ByteHash := make([]byte, 32)
	falsyValidateTransactionOrdering := func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (
		*services.ValidateTransactionsForOrderingOutput, error) {
		return &services.ValidateTransactionsForOrderingOutput{}, errors.New("Some error")
	}

	t.Run("should return error for block with incorrect protocol version", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		err := validateProtocolVersion(context.Background(), vctx)
		require.Equal(t, ErrMismatchedProtocolVersion, errors.Cause(err), "validation should fail on incorrect protocol version", err)
	})

	t.Run("should return error for block with incorrect virtual chain", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateVirtualChainId(999); err != nil {
			t.Error(err)
		}
		err := validateVirtualChainID(context.Background(), vctx)
		require.Equal(t, ErrMismatchedVirtualChainID, errors.Cause(err), "validation should fail on incorrect virtual chain", err)
	})

	t.Run("should return error for block with incorrect merkle root", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateTransactionsMerkleRootHash(empty32ByteHash); err != nil {
			t.Error(err)
		}

		err := validateTransactionBlockMerkleRoot(context.Background(), vctx)
		require.Equal(t, ErrMismatchedTxMerkleRoot, errors.Cause(err), "validation should fail on incorrect transaction root hash", err)
	})

	t.Run("should return error for block with incorrect prev block hash", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutatePrevBlockHashPtr(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err := validatePrevBlockHashPtr(context.Background(), vctx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error for block with incorrect metadata hash", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateMetadataHash(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err := validateMetadataHash(context.Background(), vctx)
		require.Equal(t, ErrMismatchedMetadataHash, errors.Cause(err), "validation should fail on incorrect metadata hash", err)
	})

	t.Run("should return error for block with failing tx ordering validation", func(t *testing.T) {
		vctx := toValidatorContext(cfg)
		vctx.txOrderValidator = falsyValidateTransactionOrdering
		err := validateTransactionOrdering(context.Background(), vctx)
		require.Equal(t, ErrIncorrectTransactionOrdering, errors.Cause(err), "validation should fail on failing tx ordering validation", err)
	})
}

func TestValidateTransactionBlock(t *testing.T) {
	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	metricFactory := metric.NewRegistry()
	cfg := config.ForConsensusContextTests(nil)
	txPool := &services.MockTransactionPool{}
	txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)

	s := NewConsensusContext(
		txPool,
		&services.MockVirtualMachine{},
		&services.MockStateStorage{},
		cfg,
		log,
		metricFactory)

	block, input := inputs(cfg)
	input.TransactionsBlock = block.TransactionsBlock
	_, err := s.ValidateTransactionsBlock(context.Background(), input)
	require.NoError(t, err, "validation should succeed on valid block")
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
