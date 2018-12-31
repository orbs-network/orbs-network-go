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
	//empty32ByteHash := make([]byte, 32)
	t.Run("should return error for results block with incorrect protocol version", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		if err := vcrx.input.ResultsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		err := validateRxProtocolVersion(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedProtocolVersion, errors.Cause(err), "validation should fail on incorrect protocol version", err)
	})

	//t.Run("should return error for block with incorrect virtual chain", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateVirtualChainId(999); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateRxVirtualChainID(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedVirtualChainID, errors.Cause(err), "validation should fail on incorrect virtual chain", err)
	//})
	//
	//t.Run("should return error for block with incorrect block height", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateBlockHeight(1); err != nil {
	//		t.Error(err)
	//	}
	//
	//	err := validateRxBlockHeight(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedBlockHeight, errors.Cause(err), "validation should fail on incorrect block height", err)
	//})
	//
	//t.Run("should return error if timestamp is not identical for transactions and results blocks", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateTimestamp(vcrx.input.TransactionsBlock.Header.Timestamp() + 1000); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateIdenticalTxRxTimestamp(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedTxRxTimestamps, errors.Cause(err), "validation should fail on different timestamps for transactions and results blocks", err)
	//})
	//
	//t.Run("should return error for block with incorrect prev block hash", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutatePrevBlockHashPtr(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateRxPrevBlockHashPtr(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	//})
	//
	//t.Run("should return error for results block which points to a different transactions block than the one it has", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateTransactionsBlockHashPtr(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateRxTxBlockPtrMatchesActualTxBlock(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect transactions block ptr", err)
	//})
	//
	//t.Run("should return error for block with incorrect receipts root hash", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateReceiptsRootHash(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateRxReceiptsRootHash(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedReceiptsRootHash, errors.Cause(err), "validation should fail on incorrect receipts root hash", err)
	//})
	//
	//t.Run("should return error for block with incorrect state diff hash", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	err := validateRxStateDiffHash(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect state diff hash", err)
	//})
	//
	//t.Run("should return error when state's pre-execution merkle root is different between the results block and state storage", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateRootHash(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	vcrx.GetStateHash = falsyGetStateHash
	//
	//	err := validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedReceipts, errors.Cause(err), "validation should fail on incorrect state diff root hash", err)
	//})
	//
	//t.Run("should return error when receipts or state merkle roots are different between calulcated execution result and those stored in block", func(t *testing.T) {
	//	vcrx := toRxValidatorContext(cfg)
	//	if err := vcrx.input.ResultsBlock.Header.MutateTransactionsBlockHashPtr(empty32ByteHash); err != nil {
	//		t.Error(err)
	//	}
	//	vcrx.processTransactionSet := falsyProcessTransactionSet
	//	err := validateExecution(context.Background(), vcrx)
	//	require.Equal(t, ErrMismatchedRxTxHashPtr, errors.Cause(err), "validation should fail on incorrect state diff root hash", err)
	//})

}

// TODO Convert to rx
func TestValidateResultsBlock(t *testing.T) {
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

	input := rxInputs(cfg)
	_, err := s.ValidateResultsBlock(context.Background(), input)
	require.NoError(t, err, "validation should succeed on valid block")
}
