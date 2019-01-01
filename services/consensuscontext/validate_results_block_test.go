package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
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

	input := &services.ValidateResultsBlockInput{
		BlockHeight:        currentBlockHeight,
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

	txBlockHashPtr := digest.CalcTransactionsBlockHash(block.TransactionsBlock)
	receiptMerkleRoot, _ := calculateReceiptsMerkleRoot(block.ResultsBlock.TransactionReceipts)
	stateDiffMerkleRoot, _ := calculateStateDiffMerkleRoot(block.ResultsBlock.ContractStateDiffs)
	preExecutionRootHash := &services.GetStateHashOutput{
		StateRootHash: empty32ByteHash,
	}

	block.ResultsBlock.Header.MutateTransactionsBlockHashPtr(txBlockHashPtr)
	block.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(receiptMerkleRoot))
	block.ResultsBlock.Header.MutateStateDiffHash(stateDiffMerkleRoot)
	block.ResultsBlock.Header.MutatePreExecutionStateRootHash(preExecutionRootHash.StateRootHash)

	return &rxValidatorContext{
		protocolVersion: cfg.ProtocolVersion(),
		virtualChainId:  cfg.VirtualChainId(),
		input: &services.ValidateResultsBlockInput{
			BlockHeight:        currentBlockHeight,
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

func NewMockGetStateHashThatReturns(stateRootHash primitives.MerkleSha256, err error) GetStateHashAdapter {
	return &mockGetStateHashAdapter{
		getStateHash: func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
			return &services.GetStateHashOutput{
				StateRootHash: stateRootHash,
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

// Mock for CalculateReceiptsMerkleRoot
type mockCalculateReceiptsMerkleRoot struct {
	calculateReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
}

func (m *mockCalculateReceiptsMerkleRoot) CalculateReceiptsMerkleRoot(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	return m.calculateReceiptsMerkleRoot(receipts)
}

func NewMockCalculateReceiptsMerkleRootThatReturns(root primitives.Sha256, err error) CalculateReceiptsMerkleRootAdapter {
	return &mockCalculateReceiptsMerkleRoot{

		calculateReceiptsMerkleRoot: func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
			return root, err
		},
	}
}

// Mock for CalculateStateDiffMerkleRoot
type mockCalculateStateDiffMerkleRoot struct {
	calculateStateDiffMerkleRoot func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func (m *mockCalculateStateDiffMerkleRoot) CalculateStateDiffMerkleRoot(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return m.calculateStateDiffMerkleRoot(stateDiffs)
}

func NewMockCalculateStateDiffMerkleRootThatReturns(root primitives.Sha256, err error) CalculateStateDiffMerkleRootAdapter {
	return &mockCalculateStateDiffMerkleRoot{
		calculateStateDiffMerkleRoot: func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
			return root, err
		},
	}
}

func TestResultsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	empty32ByteHash := make([]byte, 32)

	//falsyGetStateHash := func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	//	return &services.GetStateHashOutput{}, errors.New("Some error")
	//}
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
		err := validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutatePrevBlockHashPtr(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err = validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error for block with incorrect receipts root hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualReceiptsMerkleRoot1 := hash.CalcSha256([]byte{1})
		manualReceiptsMerkleRoot2 := hash.CalcSha256([]byte{2})
		successfulCalculateReceiptsMerkleRoot := NewMockCalculateReceiptsMerkleRootThatReturns(manualReceiptsMerkleRoot1, nil)
		vcrx.calculateReceiptsMerkleRootAdapter = successfulCalculateReceiptsMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(manualReceiptsMerkleRoot1)); err != nil {
			t.Error(err)
		}
		err := validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(manualReceiptsMerkleRoot2)); err != nil {
			t.Error(err)
		}
		err = validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect receipts root hash", err)
	})

	t.Run("should return error for block with incorrect state diff hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualStateDiffMerkleRoot1 := hash.CalcSha256([]byte{10})
		manualStateDiffMerkleRoot2 := hash.CalcSha256([]byte{20})
		successfulCalculateStateDiffMerkleRoot := NewMockCalculateStateDiffMerkleRootThatReturns(manualStateDiffMerkleRoot1, nil)
		vcrx.calculateStateDiffMerkleRootAdapter = successfulCalculateStateDiffMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot1); err != nil {
			t.Error(err)
		}
		err := validateRxStateDiffHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateRxStateDiffHash(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect state diff hash", err)
	})

	t.Run("should return error when state's pre-execution merkle root is different between the results block and state storage", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualPreExecutionStateRootHash1 := hash.CalcSha256([]byte{1})
		manualPreExecutionStateRootHash2 := hash.CalcSha256([]byte{2})

		// success case - setup the results block and GetStateHash() to return same hash
		successfulGetStateHash := NewMockGetStateHashThatReturns(primitives.MerkleSha256(manualPreExecutionStateRootHash1), nil)
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateRootHash(primitives.MerkleSha256(manualPreExecutionStateRootHash1)); err != nil {
			t.Error(err)
		}
		vcrx.getStateHashAdapter = successfulGetStateHash
		err := validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Nil(t, err, "results block holds the same pre-execution merkle root that is returned from state storage")

		// GetStateHash returns error
		errorGetStateHash := NewMockGetStateHashThatReturns(vcrx.input.ResultsBlock.Header.PreExecutionStateRootHash(), errors.New("Some error"))
		vcrx.getStateHashAdapter = errorGetStateHash
		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrGetStateHash, errors.Cause(err), "validation should fail if failed to read the pre-execution merkle root from state storage", err)

		// GetStateHash returns successfully but a mismatching hash
		vcrx.getStateHashAdapter = successfulGetStateHash
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateRootHash(primitives.MerkleSha256(manualPreExecutionStateRootHash2)); err != nil {
			t.Error(err)
		}
		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPreExecutionStateMerkleRoot, errors.Cause(err), "validation should fail if results block holds a different pre-execution merkle root than is returned from state storage", err)
	})

	t.Run("should return error when receipts or state merkle roots are different between calculated execution result and those stored in block", func(t *testing.T) {

		// TODO Add mismatching receipts and state diff checks
		vcrx := toRxValidatorContext(cfg)
		manualReceiptsMerkleRoot1 := hash.CalcSha256([]byte{1})
		manualReceiptsMerkleRoot2 := hash.CalcSha256([]byte{2})

		manualStateDiffMerkleRoot1 := hash.CalcSha256([]byte{10})
		manualStateDiffMerkleRoot2 := hash.CalcSha256([]byte{20})

		// Set expected values in results block (they will match those returned from successfulCalculateReceiptsMerkleRoot and successfulCalculateStateDiffMerkleRoot
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(manualReceiptsMerkleRoot1)); err != nil {
			t.Error(err)
		}
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot1); err != nil {
			t.Error(err)
		}

		successfulProcessTransactionSet := NewMockProcessTransactionSetThatReturns(nil)
		successfulCalculateReceiptsMerkleRoot := NewMockCalculateReceiptsMerkleRootThatReturns(manualReceiptsMerkleRoot1, nil)
		successfulCalculateStateDiffMerkleRoot := NewMockCalculateStateDiffMerkleRootThatReturns(manualStateDiffMerkleRoot1, nil)
		errorProcessTransactionSet := NewMockProcessTransactionSetThatReturns(errors.New("Some error"))
		errorCalculateReceiptsMerkleRoot := NewMockCalculateReceiptsMerkleRootThatReturns(nil, errors.New("Some error"))
		errorCalculateStateDiffMerkleRoot := NewMockCalculateStateDiffMerkleRootThatReturns(nil, errors.New("Some error"))

		// ProcessTransactionSet returns an error - returns ErrProcessTransactionSet
		vcrx.processTransactionSetAdapter = errorProcessTransactionSet
		err := validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrProcessTransactionSet, errors.Cause(err), "validation should fail if failed to execute transaction set", err)

		// CalculateReceiptsMerkleRoot returns error
		vcrx.processTransactionSetAdapter = successfulProcessTransactionSet
		vcrx.calculateReceiptsMerkleRootAdapter = errorCalculateReceiptsMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrCalculateReceiptsMerkleRoot, errors.Cause(err), "validation should fail if failed to calculate receipts merkle root", err)

		// CalculateStateDiffMerkleRoot returns error
		vcrx.calculateReceiptsMerkleRootAdapter = successfulCalculateReceiptsMerkleRoot
		vcrx.calculateStateDiffMerkleRootAdapter = errorCalculateStateDiffMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrCalculateStateDiffMerkleRoot, errors.Cause(err), "validation should fail if failed to calculate state diff merkle root", err)

		// Test the only case where everything is fine - collaborators don't return errors, and there are no mismatches
		vcrx.calculateStateDiffMerkleRootAdapter = successfulCalculateStateDiffMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Nil(t, err)

		// Now we tamper with receipts and statediff hashes in Results Block to cause mismatch errors
		// Corrupt the receipts hash
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(manualReceiptsMerkleRoot2)); err != nil {
			t.Error(err)
		}
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect post-execution receipts hash", err)

		// Restore good receipts hash
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(primitives.MerkleSha256(manualReceiptsMerkleRoot1)); err != nil {
			t.Error(err)
		}
		// Corrupt the statediff hash
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffMerkleRoot2); err != nil {
			t.Error(err)
		}
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect post-execution state diff hash", err)
	})

}
