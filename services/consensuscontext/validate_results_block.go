package consensuscontext

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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

type CalculateReceiptsMerkleRootAdapter interface {
	CalculateReceiptsMerkleRoot(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
}

type CalculateStateDiffMerkleRootAdapter interface {
	CalculateStateDiffMerkleRoot(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

type rxValidatorContext struct {
	protocolVersion                     primitives.ProtocolVersion
	virtualChainId                      primitives.VirtualChainId
	input                               *services.ValidateResultsBlockInput
	getStateHashAdapter                 GetStateHashAdapter
	processTransactionSetAdapter        ProcessTransactionSetAdapter
	calculateReceiptsMerkleRootAdapter  CalculateReceiptsMerkleRootAdapter
	calculateStateDiffMerkleRootAdapter CalculateStateDiffMerkleRootAdapter
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
	expectedReceiptsMerkleRoot := vcrx.input.ResultsBlock.Header.ReceiptsMerkleRootHash()
	calculatedReceiptMerkleRoot, err := vcrx.calculateReceiptsMerkleRootAdapter.CalculateReceiptsMerkleRoot(vcrx.input.ResultsBlock.TransactionReceipts)
	if err != nil {
		return errors.Wrapf(ErrCalculateReceiptsMerkleRoot, "ValidateResultsBlock error calculateReceiptsMerkleRoot(), %v", err)
	}
	if !bytes.Equal(expectedReceiptsMerkleRoot, []byte(calculatedReceiptMerkleRoot)) {
		return errors.Wrapf(ErrMismatchedReceiptsRootHash, "expected %v actual %v", expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot)
	}
	return nil
}

func validateRxStateDiffHash(ctx context.Context, vcrx *rxValidatorContext) error {
	expectedStateDiffMerkleRoot := vcrx.input.ResultsBlock.Header.StateDiffHash()
	calculatedStateDiffMerkleRoot, err := vcrx.calculateStateDiffMerkleRootAdapter.CalculateStateDiffMerkleRoot(vcrx.input.ResultsBlock.ContractStateDiffs)
	if err != nil {
		return errors.Wrapf(ErrCalculateStateDiffMerkleRoot, "ValidateResultsBlock error calculateStateDiffMerkleRoot(), %v", err)
	}
	if !bytes.Equal(expectedStateDiffMerkleRoot, []byte(calculatedStateDiffMerkleRoot)) {
		return errors.Wrapf(ErrMismatchedStateDiffHash, "expected %v actual %v", expectedStateDiffMerkleRoot, calculatedStateDiffMerkleRoot)
	}
	return nil
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
	processTxsOut, err := vcrx.processTransactionSetAdapter.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{ // TODO wrap with adapter
		CurrentBlockHeight:    vcrx.input.TransactionsBlock.Header.BlockHeight(),
		CurrentBlockTimestamp: vcrx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions:    vcrx.input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return errors.Wrapf(ErrProcessTransactionSet, "ValidateResultsBlock.validateExecution() error ProcessTransactionSet")
	}
	// Compare the receipts merkle root hash to the one in the block.
	expectedReceiptsMerkleRoot := vcrx.input.ResultsBlock.Header.ReceiptsMerkleRootHash()
	calculatedReceiptMerkleRoot, err := vcrx.calculateReceiptsMerkleRootAdapter.CalculateReceiptsMerkleRoot(processTxsOut.TransactionReceipts) // TODO wrap with adapter
	if err != nil {
		return errors.Wrapf(ErrCalculateReceiptsMerkleRoot, "ValidateResultsBlock error ProcessTransactionSet calculateReceiptsMerkleRoot")
	}
	if !bytes.Equal(expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot) {
		return errors.Wrapf(ErrMismatchedReceiptsRootHash, "ValidateResultsBlock error receipt merkleRoot in header does not match processed txs receipts")
	}

	// Compare the state diff hash to the one in the block (supports only deterministic execution).
	expectedStateDiffMerkleRoot := vcrx.input.ResultsBlock.Header.RawStateDiffHash()
	calculatedStateDiffMerkleRoot, err := vcrx.calculateStateDiffMerkleRootAdapter.CalculateStateDiffMerkleRoot(processTxsOut.ContractStateDiffs) // TODO wrap with adapter
	if err != nil {
		return errors.Wrapf(ErrCalculateStateDiffMerkleRoot, "ValidateResultsBlock error ProcessTransactionSet calculateStateDiffMerkleRoot")
	}
	if !bytes.Equal(expectedStateDiffMerkleRoot, calculatedStateDiffMerkleRoot) {
		return errors.Wrapf(ErrMismatchedStateDiffHash, "expected %v actual %v", expectedStateDiffMerkleRoot, calculatedStateDiffMerkleRoot)
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

type realCalculateReceiptsMerkleRootAdapter struct {
	calculateReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
}

func (r *realCalculateReceiptsMerkleRootAdapter) CalculateReceiptsMerkleRoot(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	return r.calculateReceiptsMerkleRoot(receipts)
}
func NewRealCalculateReceiptsMerkleRootAdapter(f func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)) CalculateReceiptsMerkleRootAdapter {
	return &realCalculateReceiptsMerkleRootAdapter{
		calculateReceiptsMerkleRoot: f,
	}
}

type realCalculateStateDiffMerkleRootAdapter struct {
	calculateStateDiffMerkleRoot func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

func (r *realCalculateStateDiffMerkleRootAdapter) CalculateStateDiffMerkleRoot(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	return r.CalculateStateDiffMerkleRoot(stateDiffs)
}
func NewRealCalculateStateDiffMerkleRootAdapter(f func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)) CalculateStateDiffMerkleRootAdapter {
	return &realCalculateStateDiffMerkleRootAdapter{
		calculateStateDiffMerkleRoot: f,
	}
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {

	vcrx := &rxValidatorContext{
		protocolVersion:                     s.config.ProtocolVersion(),
		virtualChainId:                      s.config.VirtualChainId(),
		input:                               input,
		getStateHashAdapter:                 NewRealGetStateHashAdapter(s.stateStorage.GetStateHash),
		processTransactionSetAdapter:        NewRealProcessTransactionSetAdapter(s.virtualMachine.ProcessTransactionSet),
		calculateReceiptsMerkleRootAdapter:  NewRealCalculateReceiptsMerkleRootAdapter(calculateReceiptsMerkleRoot),
		calculateStateDiffMerkleRootAdapter: NewRealCalculateStateDiffMerkleRootAdapter(calculateStateDiffMerkleRoot),
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
