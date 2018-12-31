package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
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
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.AddressForEd25519SignerForTests(6)).Build()
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
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.AddressForEd25519SignerForTests(6)).Build()
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

func TestResultsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	empty32ByteHash := make([]byte, 32)

	truthyGetStateHash := func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
		return &services.GetStateHashOutput{
			StateRootHash: empty32ByteHash,
		}, nil
	}
	//falsyGetStateHash := func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	//	return &services.GetStateHashOutput{}, errors.New("Some error")
	//}
	truthyProcessTransactionSet := func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
		return &services.ProcessTransactionSetOutput{
			TransactionReceipts: nil,
			ContractStateDiffs:  nil,
		}, nil
	}
	falsyProcessTransactionSet := func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
		return &services.ProcessTransactionSetOutput{}, errors.New("Some error")
	}

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
		err := validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err = validateRxReceiptsRootHash(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect receipts root hash", err)
	})

	t.Run("should return error for block with incorrect state diff hash", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		err := validateRxStateDiffHash(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(empty32ByteHash); err != nil {
			t.Error(err)
		}
		err = validateRxStateDiffHash(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect state diff hash", err)
	})

	t.Run("should return error when state's pre-execution merkle root is different between the results block and state storage", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		vcrx.getStateHash = truthyGetStateHash
		err := validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateRootHash(empty32ByteHash); err != nil {
			t.Error(err)
		}
		//vcrx.getStateHash = falsyGetStateHash

		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPreExecutionStateMerkleRoot, errors.Cause(err), "validation should fail on incorrect pre-execution state root hash", err)
	})

	t.Run("should return error when receipts or state merkle roots are different between calculated execution result and those stored in block", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		vcrx.processTransactionSet = truthyProcessTransactionSet
		err := validateExecution(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateTransactionsBlockHashPtr(empty32ByteHash); err != nil {
			t.Error(err)
		}
		vcrx.processTransactionSet = falsyProcessTransactionSet
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPreExecutionStateMerkleRoot, errors.Cause(err), "validation should fail on incorrect post-execution receipts or state hash", err)
	})

}
