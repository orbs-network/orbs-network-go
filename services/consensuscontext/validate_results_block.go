package consensuscontext

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type rxValidator func(ctx context.Context, vcrx *rxValidatorContext) error

type GetStateHashAdapter interface {
	GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)
}

type ProcessTransactionSetAdapter interface {
	ProcessTransactionSet(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)
}

type rxValidatorContext struct {
	protocolVersion               primitives.ProtocolVersion
	virtualChainId                primitives.VirtualChainId
	input                         *services.ValidateResultsBlockInput
	getStateHashAdapter           GetStateHashAdapter
	processTransactionSetAdapter  ProcessTransactionSetAdapter
	calcReceiptsMerkleRootAdapter digest.CalcReceiptsMerkleRootAdapter
	calcStateDiffHashAdapter      digest.CalcStateDiffHashAdapter
}

func validateRxProtocolVersion(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedProtocolVersion := vcrx.protocolVersion
	checkedProtocolVersion := vcrx.input.ResultsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}
	return nil
}

func validateRxVirtualChainID(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedVirtualChainId := vcrx.virtualChainId
	checkedVirtualChainId := vcrx.input.ResultsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != expectedVirtualChainId {
		return errors.Wrapf(ErrMismatchedVirtualChainID, "expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}
	return nil
}

func validateRxBlockHeight(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedBlockHeight := vcrx.input.CurrentBlockHeight
	checkedBlockHeight := vcrx.input.ResultsBlock.Header.BlockHeight()
	if checkedBlockHeight != expectedBlockHeight {
		return errors.Wrapf(ErrMismatchedBlockHeight, "expected %v actual %v", expectedBlockHeight, checkedBlockHeight)
	}
	txBlockHeight := vcrx.input.TransactionsBlock.Header.BlockHeight()
	if checkedBlockHeight != txBlockHeight {
		return errors.Wrapf(ErrMismatchedTxRxBlockHeight, "txBlock %v rxBlock %v", txBlockHeight, checkedBlockHeight)
	}
	return nil
}

func validateRxTxBlockPtrMatchesActualTxBlock(ctx context.Context, vcrx *rxValidatorContext) error {
	txBlockHashPtr := vcrx.input.ResultsBlock.Header.TransactionsBlockHashPtr()
	expectedTxBlockHashPtr := digest.CalcTransactionsBlockHash(vcrx.input.TransactionsBlock)
	if !bytes.Equal(txBlockHashPtr, expectedTxBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedTxHashPtrToActualTxBlock, "expected %v actual %v", expectedTxBlockHashPtr, txBlockHashPtr)
	}
	return nil
}

func validateIdenticalTxRxTimestamp(ctx context.Context, vcrx *rxValidatorContext) error {
	txTimestamp := vcrx.input.TransactionsBlock.Header.Timestamp()
	rxTimestamp := vcrx.input.ResultsBlock.Header.Timestamp()
	if rxTimestamp != txTimestamp {
		return errors.Wrapf(ErrMismatchedTxRxTimestamps, "txTimestamp %v rxTimestamp %v", txTimestamp, rxTimestamp)
	}
	return nil
}

func validateRxPrevBlockHashPtr(ctx context.Context, vcrx *rxValidatorContext) error {
	prevBlockHashPtr := vcrx.input.ResultsBlock.Header.PrevBlockHashPtr()
	expectedPrevBlockHashPtr := vcrx.input.PrevBlockHash
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedPrevBlockHash, "expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}
	return nil
}

func validateRxReceiptsRootHash(ctx context.Context, vcrx *rxValidatorContext) error {
	return validators.ValidateReceiptsMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock:      vcrx.input.TransactionsBlock,
		ResultsBlock:           vcrx.input.ResultsBlock,
		CalcReceiptsMerkleRoot: vcrx.calcReceiptsMerkleRootAdapter.CalcReceiptsMerkleRoot,
	})
}

func validateRxStateDiffHash(ctx context.Context, vcrx *rxValidatorContext) error {
	return validators.ValidateResultsBlockStateDiffHash(&validators.BlockValidatorContext{
		TransactionsBlock: vcrx.input.TransactionsBlock,
		ResultsBlock:      vcrx.input.ResultsBlock,
		CalcStateDiffHash: vcrx.calcStateDiffHashAdapter.CalcStateDiffHash,
	})
}

func validatePreExecutionStateMerkleRoot(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedPreExecutionMerkleRoot := vcrx.input.ResultsBlock.Header.PreExecutionStateMerkleRootHash()
	getStateHashOut, err := vcrx.getStateHashAdapter.GetStateHash(ctx, &services.GetStateHashInput{
		BlockHeight: vcrx.input.ResultsBlock.Header.BlockHeight() - 1,
	})
	if err != nil {
		return errors.Wrapf(ErrGetStateHash, "ValidateResultsBlock.validatePreExecutionStateMerkleRoot() error GetStateHash(), %v", err)
	}
	if !bytes.Equal(expectedPreExecutionMerkleRoot, getStateHashOut.StateMerkleRootHash) {
		return errors.Wrapf(ErrMismatchedPreExecutionStateMerkleRoot, "expected %v actual %v", expectedPreExecutionMerkleRoot, getStateHashOut.StateMerkleRootHash)
	}
	return nil
}

func validateExecution(ctx context.Context, vcrx *rxValidatorContext) error {
	//Validate transaction execution
	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet creating receipts and state diff. Using the provided header timestamp as a reference timestamp.
	processTxsOut, err := vcrx.processTransactionSetAdapter.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		CurrentBlockHeight:    vcrx.input.TransactionsBlock.Header.BlockHeight(),
		CurrentBlockTimestamp: vcrx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions:    vcrx.input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return errors.Wrapf(ErrProcessTransactionSet, "ValidateResultsBlock.validateExecution() error ProcessTransactionSet")
	}
	// Compare the receipts merkle root hash to the one in the block.
	expectedReceiptsMerkleRoot := vcrx.input.ResultsBlock.Header.ReceiptsMerkleRootHash()
	calculatedReceiptMerkleRoot, err := vcrx.calcReceiptsMerkleRootAdapter.CalcReceiptsMerkleRoot(processTxsOut.TransactionReceipts)
	if err != nil {
		return errors.Wrapf(validators.ErrCalcReceiptsMerkleRoot, "ValidateResultsBlock error ProcessTransactionSet calculateReceiptsMerkleRoot")
	}
	if !bytes.Equal(expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot) {
		return errors.Wrapf(validators.ErrMismatchedReceiptsRootHash, "expected %v actual %v", expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot)
	}

	// Compare the state diff hash to the one in the block (supports only deterministic execution).
	expectedStateDiffHash := vcrx.input.ResultsBlock.Header.StateDiffHash()
	calculatedStateDiffHash, err := vcrx.calcStateDiffHashAdapter.CalcStateDiffHash(processTxsOut.ContractStateDiffs)
	if err != nil {
		return errors.Wrapf(validators.ErrCalcStateDiffHash, "ValidateResultsBlock error ProcessTransactionSet calculateStateDiffHash")
	}
	if !bytes.Equal(expectedStateDiffHash, calculatedStateDiffHash) {
		return errors.Wrapf(validators.ErrMismatchedStateDiffHash, "expected %v actual %v", expectedStateDiffHash, calculatedStateDiffHash)
	}

	return nil
}

type realGetStateHashAdapter struct {
	getStateHash func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)
}

func (r *realGetStateHashAdapter) GetStateHash(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	return r.getStateHash(ctx, input)
}
func NewRealGetStateHashAdapter(f func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error)) GetStateHashAdapter {
	return &realGetStateHashAdapter{
		getStateHash: f,
	}
}

type realProcessTransactionSetAdapter struct {
	processTransactionSet func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)
}

func (r *realProcessTransactionSetAdapter) ProcessTransactionSet(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	return r.processTransactionSet(ctx, input)
}
func NewRealProcessTransactionSetAdapter(f func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error)) ProcessTransactionSetAdapter {
	return &realProcessTransactionSetAdapter{
		processTransactionSet: f,
	}
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {

	vcrx := &rxValidatorContext{
		protocolVersion:               s.config.ProtocolVersion(),
		virtualChainId:                s.config.VirtualChainId(),
		input:                         input,
		getStateHashAdapter:           NewRealGetStateHashAdapter(s.stateStorage.GetStateHash),
		processTransactionSetAdapter:  NewRealProcessTransactionSetAdapter(s.virtualMachine.ProcessTransactionSet),
		calcReceiptsMerkleRootAdapter: digest.NewRealCalcReceiptsMerkleRootAdapter(digest.CalcReceiptsMerkleRoot),
		calcStateDiffHashAdapter:      digest.NewRealCalcStateDiffHashAdapter(digest.CalcStateDiffHash),
	}

	validators := []rxValidator{
		validateRxProtocolVersion,
		validateRxVirtualChainID,
		validateRxBlockHeight,
		validateRxTxBlockPtrMatchesActualTxBlock,
		validateIdenticalTxRxTimestamp,
		validateRxPrevBlockHashPtr,
		validateRxReceiptsRootHash,
		validateRxStateDiffHash,
		validatePreExecutionStateMerkleRoot,
		validateExecution,
	}

	for _, v := range validators {
		if err := v(ctx, vcrx); err != nil {
			return &services.ValidateResultsBlockOutput{}, err
		}
	}
	return &services.ValidateResultsBlockOutput{}, nil
}
