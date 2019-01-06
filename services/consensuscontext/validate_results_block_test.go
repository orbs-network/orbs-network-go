package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testDigest "github.com/orbs-network/orbs-network-go/test/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func rxInputs(cfg config.ConsensusContextConfig) *services.ValidateResultsBlockInput {

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

	input := &services.ValidateResultsBlockInput{
		CurrentBlockHeight: currentBlockHeight,
		TransactionsBlock:  block.TransactionsBlock,
		ResultsBlock:       block.ResultsBlock,
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return input
}

func toRxValidatorContext(cfg config.ConsensusContextConfig) *rxValidatorContext {

	empty32ByteHash := make([]byte, 32)
	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := digest.CalcTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
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

	txBlockHashPtr := digest.CalcTransactionsBlockHash(block.TransactionsBlock)
	receiptMerkleRoot, _ := digest.CalcReceiptsMerkleRoot(block.ResultsBlock.TransactionReceipts)
	stateDiffMerkleRoot, _ := digest.CalcStateDiffMerkleRoot(block.ResultsBlock.ContractStateDiffs)
	preExecutionRootHash := &services.GetStateHashOutput{
		StateMerkleRootHash: empty32ByteHash,
	}

	block.ResultsBlock.Header.MutateTransactionsBlockHashPtr(txBlockHashPtr)
	block.ResultsBlock.Header.MutateReceiptsMerkleRootHash(receiptMerkleRoot)
	block.ResultsBlock.Header.MutateStateDiffHash(stateDiffMerkleRoot)
	block.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(preExecutionRootHash.StateMerkleRootHash)

	return &rxValidatorContext{
		protocolVersion: cfg.ProtocolVersion(),
		virtualChainId:  cfg.VirtualChainId(),
		input: &services.ValidateResultsBlockInput{
			CurrentBlockHeight: currentBlockHeight,
			ResultsBlock:       block.ResultsBlock,
			PrevBlockHash:      validPrevBlockHash,
			TransactionsBlock:  block.TransactionsBlock,
			PrevBlockTimestamp: validPrevBlockTimestamp,
		},
	}

}

// Mock for GetStateHash
type mockGetStateHashAdapter struct {
	getStateHash func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)
}

func NewMockGetStateHashThatReturns(stateRootHash primitives.Sha256, err error) GetStateHashAdapter {
	return &mockGetStateHashAdapter{
		getStateHash: func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
			return &services.GetStateHashOutput{
				StateMerkleRootHash: stateRootHash,
			}, err
		},
	}
}
func (m *mockGetStateHashAdapter) GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	return m.getStateHash(ctx, input)
}

// Mock for ProcessTransactionSet
type mockProcessTransactionSet struct {
	processTransactionSet func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)
}

func (m *mockProcessTransactionSet) ProcessTransactionSet(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	return m.processTransactionSet(ctx, input)
}

func NewMockProcessTransactionSetThatReturns(err error) ProcessTransactionSetAdapter {

	someEmptyTxSetThatWeReturnOnlyToPreventErrors := &services.ProcessTransactionSetOutput{
		TransactionReceipts: nil,
		ContractStateDiffs:  nil,
	}

	return &mockProcessTransactionSet{
		processTransactionSet: func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
			return someEmptyTxSetThatWeReturnOnlyToPreventErrors, err
		},
	}
}

func TestResultsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	empty32ByteHash := make([]byte, 32)
	t.Run("should return error for results block with incorrect protocol version", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxProtocolVersion(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		err = validateRxProtocolVersion(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedProtocolVersion, errors.Cause(err), "validation should fail on incorrect protocol version in results block", err)
	})

	t.Run("should return error for block with incorrect virtual chain", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxVirtualChainID(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateVirtualChainId(999); err != nil {
			t.Error(err)
		}
		err = validateRxVirtualChainID(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedVirtualChainID, errors.Cause(err), "validation should fail on incorrect virtual chain in results block", err)
	})

	t.Run("should return error for results block with incorrect block height", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxBlockHeight(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateBlockHeight(1); err != nil {
			t.Error(err)
		}

		err = validateRxBlockHeight(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedBlockHeight, errors.Cause(err), "validation should fail on incorrect block height", err)
	})

	t.Run("should return error for different height between transactions and results blocks", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxBlockHeight(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.TransactionsBlock.Header.MutateBlockHeight(1); err != nil {
			t.Error(err)
		}

		err = validateRxBlockHeight(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedTxRxBlockHeight, errors.Cause(err), "validation should fail on different height between transactions and results blocks", err)
	})

	t.Run("should return error for results block which points to a different transactions block than the one it has", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxTxBlockPtrMatchesActualTxBlock(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateTransactionsBlockHashPtr(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err = validateRxTxBlockPtrMatchesActualTxBlock(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedTxHashPtrToActualTxBlock, errors.Cause(err), "validation should fail on incorrect transactions block ptr", err)
	})

	t.Run("should return error if timestamp is not identical for transactions and results blocks", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateIdenticalTxRxTimestamp(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateTimestamp(vcrx.input.TransactionsBlock.Header.Timestamp() + 1000); err != nil {
			t.Error(err)
		}
		err = validateIdenticalTxRxTimestamp(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedTxRxTimestamps, errors.Cause(err), "validation should fail on different timestamps for transactions and results blocks", err)
	})

	t.Run("should return error for block with incorrect prev block hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		hash2 := hash.CalcSha256([]byte{2})
		err := validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutatePrevBlockHashPtr(hash2); err != nil {
			t.Error(err)
		}
		err = validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error for block with incorrect receipts root hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualReceiptsMerkleRoot1 := hash.CalcSha256([]byte{1})
		manualReceiptsMerkleRoot2 := hash.CalcSha256([]byte{2})
		successfulCalculateReceiptsMerkleRoot := testDigest.NewMockCalcReceiptsMerkleRootThatReturns(manualReceiptsMerkleRoot1, nil)
		vcrx.calcReceiptsMerkleRootAdapter = successfulCalculateReceiptsMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot1); err != nil {
			t.Error(err)
		}
		err := validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Equal(t, validators.ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect receipts root hash", err)
	})

	t.Run("should return error for block with incorrect state diff hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualStateDiffMerkleRoot1 := hash.CalcSha256([]byte{10})
		manualStateDiffMerkleRoot2 := hash.CalcSha256([]byte{20})
		successfulCalcStateDiffMerkleRoot := testDigest.NewMockCalcStateDiffMerkleRootThatReturns(manualStateDiffMerkleRoot1, nil)
		vcrx.calcStateDiffMerkleRootAdapter = successfulCalcStateDiffMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot1); err != nil {
			t.Error(err)
		}
		err := validateRxStateDiffHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateRxStateDiffHash(context.Background(), vcrx)
		require.Equal(t, validators.ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect state diff hash", err)
	})

	t.Run("should return error when state's pre-execution merkle root is different between the results block and state storage", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualPreExecutionStateMerkleRootHash1 := hash.CalcSha256([]byte{1})
		manualPreExecutionStateMerkleRootHash2 := hash.CalcSha256([]byte{2})

		// success case - setup the results block and GetStateHash() to return same hash
		successfulGetStateHash := NewMockGetStateHashThatReturns(manualPreExecutionStateMerkleRootHash1, nil)
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(manualPreExecutionStateMerkleRootHash1); err != nil {
			t.Error(err)
		}
		vcrx.getStateHashAdapter = successfulGetStateHash
		err := validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Nil(t, err, "results block holds the same pre-execution merkle root that is returned from state storage")

		// GetStateHash returns error
		errorGetStateHash := NewMockGetStateHashThatReturns(vcrx.input.ResultsBlock.Header.PreExecutionStateMerkleRootHash(), errors.New("Some error"))
		vcrx.getStateHashAdapter = errorGetStateHash
		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrGetStateHash, errors.Cause(err), "validation should fail if failed to read the pre-execution merkle root from state storage", err)

		// GetStateHash returns successfully but a mismatching hash
		vcrx.getStateHashAdapter = successfulGetStateHash
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(manualPreExecutionStateMerkleRootHash2); err != nil {
			t.Error(err)
		}
		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPreExecutionStateMerkleRoot, errors.Cause(err), "validation should fail if results block holds a different pre-execution merkle root than is returned from state storage", err)
	})

	t.Run("should return error when receipts or state merkle roots are different between calculated execution result and those stored in block", func(t *testing.T) {

		vcrx := toRxValidatorContext(cfg)
		manualReceiptsMerkleRoot1 := hash.CalcSha256([]byte{1})
		manualReceiptsMerkleRoot2 := hash.CalcSha256([]byte{2})

		manualStateDiffMerkleRoot1 := hash.CalcSha256([]byte{10})
		manualStateDiffMerkleRoot2 := hash.CalcSha256([]byte{20})

		// Set expected values in results block (they will match those returned from successfulCalculateReceiptsMerkleRoot and successfulCalculateStateDiffMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot1); err != nil {
			t.Error(err)
		}
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot1); err != nil {
			t.Error(err)
		}

		successfulProcessTransactionSet := NewMockProcessTransactionSetThatReturns(nil)
		successfulCalcReceiptsMerkleRoot := testDigest.NewMockCalcReceiptsMerkleRootThatReturns(manualReceiptsMerkleRoot1, nil)
		successfulCalcStateDiffMerkleRoot := testDigest.NewMockCalcStateDiffMerkleRootThatReturns(manualStateDiffMerkleRoot1, nil)
		errorProcessTransactionSet := NewMockProcessTransactionSetThatReturns(errors.New("Some error"))
		errorCalcReceiptsMerkleRoot := testDigest.NewMockCalcReceiptsMerkleRootThatReturns(nil, errors.New("Some error"))
		errorCalcStateDiffMerkleRoot := testDigest.NewMockCalcStateDiffMerkleRootThatReturns(nil, errors.New("Some error"))

		// ProcessTransactionSet returns an error - returns ErrProcessTransactionSet
		vcrx.processTransactionSetAdapter = errorProcessTransactionSet
		err := validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrProcessTransactionSet, errors.Cause(err), "validation should fail if failed to execute transaction set", err)

		// CalcReceiptsMerkleRoot returns error
		vcrx.processTransactionSetAdapter = successfulProcessTransactionSet
		vcrx.calcReceiptsMerkleRootAdapter = errorCalcReceiptsMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrCalcReceiptsMerkleRoot, errors.Cause(err), "validation should fail if failed to calculate receipts merkle root", err)

		// CalcStateDiffMerkleRoot returns error
		vcrx.calcReceiptsMerkleRootAdapter = successfulCalcReceiptsMerkleRoot
		vcrx.calcStateDiffMerkleRootAdapter = errorCalcStateDiffMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrCalcStateDiffMerkleRoot, errors.Cause(err), "validation should fail if failed to calculate state diff merkle root", err)

		// Test the only case where everything is fine - collaborators don't return errors, and there are no mismatches
		vcrx.calcStateDiffMerkleRootAdapter = successfulCalcStateDiffMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Nil(t, err)

		// Now we tamper with receipts and statediff hashes in Results Block to cause mismatch errors
		// Corrupt the receipts hash
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect post-execution receipts hash", err)

		// Restore good receipts hash
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot1); err != nil {
			t.Error(err)
		}
		// Corrupt the statediff hash
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect post-execution state diff hash", err)
	})

}
