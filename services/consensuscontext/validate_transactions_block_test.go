package consensuscontext

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testValidators "github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func txInputs(cfg config.ConsensusContextConfig) *services.ValidateTransactionsBlockInput {

	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := digest.CalcTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
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
		TransactionsBlock:  block.TransactionsBlock,
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return input
}

func toTxValidatorContext(cfg config.ConsensusContextConfig) *txValidatorContext {

	block := testValidators.BuildValidTestBlock()
	prevBlockHashCopy := make([]byte, 32)
	copy(prevBlockHashCopy, block.TransactionsBlock.Header.PrevBlockHashPtr())

	input := &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
		TransactionsBlock:  block.TransactionsBlock, // fill in each test
		PrevBlockHash:      prevBlockHashCopy,
		//PrevBlockTimestamp: block.TransactionsBlock.,
	}

	return &txValidatorContext{
		protocolVersion:        cfg.ProtocolVersion(),
		virtualChainId:         cfg.VirtualChainId(),
		allowedTimestampJitter: cfg.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
	}
}

func TestTransactionsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	hash2 := hash.CalcSha256([]byte{2})
	falsyValidateTransactionOrdering := func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (
		*services.ValidateTransactionsForOrderingOutput, error) {
		return &services.ValidateTransactionsForOrderingOutput{}, errors.New("Some error")
	}

	t.Run("should return error for transaction block with incorrect protocol version", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		err := validateTxProtocolVersion(context.Background(), vctx)
		require.Equal(t, ErrMismatchedProtocolVersion, errors.Cause(err), "validation should fail on incorrect protocol version", err)
	})

	t.Run("should return error for transaction block with incorrect virtual chain", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateVirtualChainId(999); err != nil {
			t.Error(err)
		}
		err := validateTxVirtualChainID(context.Background(), vctx)
		require.Equal(t, ErrMismatchedVirtualChainID, errors.Cause(err), "validation should fail on incorrect virtual chain", err)
	})

	t.Run("should return error for transaction block with incorrect merkle root", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateTransactionsMerkleRootHash(hash2); err != nil {
			t.Error(err)
		}

		err := validateTransactionsBlockMerkleRoot(context.Background(), vctx)
		require.Equal(t, validators.ErrMismatchedTxMerkleRoot, errors.Cause(err), "validation should fail on incorrect transaction root hash", err)
	})

	t.Run("should return error for transaction block with incorrect block height", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateBlockHeight(1); err != nil {
			t.Error(err)
		}

		err := validateTxBlockHeight(context.Background(), vctx)
		require.Equal(t, ErrMismatchedBlockHeight, errors.Cause(err), "validation should fail on incorrect block height", err)
	})

	t.Run("should return error for transaction block with incorrect prev block hash", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutatePrevBlockHashPtr(hash2); err != nil {
			t.Error(err)
		}
		err := validateTxPrevBlockHashPtr(context.Background(), vctx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error for transaction block with incorrect metadata hash", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateMetadataHash(hash2); err != nil {
			t.Error(err)
		}
		err := validateTransactionsBlockMetadataHash(context.Background(), vctx)
		require.Equal(t, validators.ErrMismatchedMetadataHash, errors.Cause(err), "validation should fail on incorrect metadata hash", err)
	})

	t.Run("should return error for transaction block with failing tx ordering validation", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		vctx.txOrderValidator = falsyValidateTransactionOrdering
		err := validateTxTransactionOrdering(context.Background(), vctx)
		require.Equal(t, ErrIncorrectTransactionOrdering, errors.Cause(err), "validation should fail on failing tx ordering validation", err)
	})
}

func TestValidateTransactionsBlock(t *testing.T) {
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

	input := txInputs(cfg)
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
