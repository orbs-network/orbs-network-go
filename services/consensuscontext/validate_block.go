package consensuscontext

import (
	"bytes"
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

var ErrMismatchedProtocolVersion = errors.New("mismatched protocol version")
var ErrMismatchedVirtualChainID = errors.New("mismatched virtual chain ID")
var ErrMismatchedBlockHeight = errors.New("mismatched block height")
var ErrMismatchedPrevBlockHash = errors.New("mismatched previous block hash")
var ErrInvalidBlockTimestamp = errors.New("invalid current block timestamp")
var ErrMismatchedTxMerkleRoot = errors.New("mismatched transactions merkle root")
var ErrMismatchedMetadataHash = errors.New("mismatched metadata hash")
var ErrIncorrectTransactionOrdering = errors.New("incorrect transaction ordering")

type validator func(ctx context.Context, vctx *validatorContext) error

type validatorContext struct {
	protocolVersion        primitives.ProtocolVersion
	virtualChainId         primitives.VirtualChainId
	allowedTimestampJitter time.Duration
	input                  *services.ValidateTransactionsBlockInput
	txOrderValidator       func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error)
}

func validateProtocolVersion(ctx context.Context, vctx *validatorContext) error {
	expectedProtocolVersion := vctx.protocolVersion
	checkedProtocolVersion := vctx.input.TransactionsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}
	return nil
}

func validateVirtualChainID(ctx context.Context, vctx *validatorContext) error {
	expectedVirtualChainId := vctx.virtualChainId
	checkedVirtualChainId := vctx.input.TransactionsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != vctx.virtualChainId {
		return errors.Wrapf(ErrMismatchedVirtualChainID, "expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}
	return nil
}

func validateBlockHeight(ctx context.Context, vctx *validatorContext) error {
	checkedBlockHeight := vctx.input.TransactionsBlock.Header.BlockHeight()
	expectedBlockHeight := vctx.input.BlockHeight
	if checkedBlockHeight != expectedBlockHeight {
		return ErrMismatchedBlockHeight
	}
	return nil
}

func validatePrevBlockHashPtr(ctx context.Context, vctx *validatorContext) error {
	expectedPrevBlockHashPtr := vctx.input.PrevBlockHash
	prevBlockHashPtr := vctx.input.TransactionsBlock.Header.PrevBlockHashPtr()
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedPrevBlockHash, "expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}
	return nil
}

func validateTransactionBlockTimestamp(ctx context.Context, vctx *validatorContext) error {
	prevBlockTimestamp := vctx.input.PrevBlockTimestamp
	currentBlockTimestamp := vctx.input.TransactionsBlock.Header.Timestamp()
	allowedTimestampJitter := vctx.allowedTimestampJitter
	now := time.Now()
	if !isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter) {
		return errors.Wrapf(ErrInvalidBlockTimestamp, "currentTimestamp %v prevTimestamp %v now %v allowed jitter %v",
			currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter)
	}
	return nil
}

func validateTransactionBlockMerkleRoot(ctx context.Context, vctx *validatorContext) error {
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	txMerkleRoot := vctx.input.TransactionsBlock.Header.TransactionsMerkleRootHash()
	if expectedTxMerkleRoot, err := calculateTransactionsMerkleRoot(vctx.input.TransactionsBlock.SignedTransactions); err != nil {
		return err
	} else if !bytes.Equal(txMerkleRoot, expectedTxMerkleRoot) {
		return errors.Wrapf(ErrMismatchedTxMerkleRoot, "expected %v actual %v", expectedTxMerkleRoot, txMerkleRoot)
	}
	return nil
}

func validateMetadataHash(ctx context.Context, vctx *validatorContext) error {
	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(vctx.input.TransactionsBlock.Metadata)
	metadataHash := vctx.input.TransactionsBlock.Header.MetadataHash()
	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
		return errors.Wrapf(ErrMismatchedMetadataHash, "expected %v actual %v", expectedMetaDataHash, metadataHash)
	}
	return nil
}

func validateTransactionOrdering(ctx context.Context, vctx *validatorContext) error {
	validationInput := &services.ValidateTransactionsForOrderingInput{
		BlockHeight:        vctx.input.TransactionsBlock.Header.BlockHeight(),
		BlockTimestamp:     vctx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions: vctx.input.TransactionsBlock.SignedTransactions,
	}
	_, err := vctx.txOrderValidator(ctx, validationInput)
	if err != nil {
		return ErrIncorrectTransactionOrdering
	}
	return nil
}

func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {

	vctx := &validatorContext{
		protocolVersion:        s.config.ProtocolVersion(),
		virtualChainId:         s.config.VirtualChainId(),
		allowedTimestampJitter: s.config.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
		txOrderValidator:       s.transactionPool.ValidateTransactionsForOrdering,
	}

	validators := []validator{
		validateProtocolVersion,
		validateVirtualChainID,
		validateBlockHeight,
		validatePrevBlockHashPtr,
		validateTransactionBlockTimestamp,
		validateTransactionBlockMerkleRoot,
		validateMetadataHash,
		validateTransactionOrdering,
	}

	for _, v := range validators {
		if err := v(ctx, vctx); err != nil {
			return &services.ValidateTransactionsBlockOutput{}, err
		}
	}
	return &services.ValidateTransactionsBlockOutput{}, nil
}

func isValidBlockTimestamp(currentBlockTimestamp primitives.TimestampNano, prevBlockTimestamp primitives.TimestampNano, now time.Time, allowedTimestampJitter time.Duration) bool {

	// TODO v1 decide on this: No, we do not handle gracefully dates before 1970
	if now.UnixNano() < 0 {
		panic("we don't handle dates before 1970")
	}

	if prevBlockTimestamp >= currentBlockTimestamp {
		return false
	}
	if uint64(currentBlockTimestamp) > uint64(now.Add(allowedTimestampJitter).UnixNano()) {
		return false
	}

	if uint64(currentBlockTimestamp) < uint64(now.Add(-allowedTimestampJitter).UnixNano()) {
		return false
	}
	return true
}

func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {

	err := ValidateResultsBlockInternal(ctx, input,
		s.config.ProtocolVersion(), s.config.VirtualChainId(),
		s.stateStorage.GetStateHash,
		s.virtualMachine.ProcessTransactionSet)
	return &services.ValidateResultsBlockOutput{}, err

}

func ValidateResultsBlockInternal(ctx context.Context, input *services.ValidateResultsBlockInput,
	expectedProtocolVersion primitives.ProtocolVersion,
	expectedVirtualChainId primitives.VirtualChainId,
	getStateHash func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error),
	processTransactionSet func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error),
) error {
	fmt.Println("ValidateResultsBlock ", ctx, input)

	checkedHeader := input.ResultsBlock.Header
	blockProtocolVersion := checkedHeader.ProtocolVersion()
	blockVirtualChainId := checkedHeader.VirtualChainId()

	if blockProtocolVersion != expectedProtocolVersion {
		return fmt.Errorf("incorrect protocol version: expected %v but block has %v", expectedProtocolVersion, blockProtocolVersion)
	}

	if blockVirtualChainId != expectedVirtualChainId {
		return fmt.Errorf("incorrect virtual chain ID: expected %v but block has %v", expectedVirtualChainId, blockVirtualChainId)
	}

	if input.BlockHeight != checkedHeader.BlockHeight() {
		return fmt.Errorf("mismatching blockHeight: input %v checkedHeader %v", input.BlockHeight, checkedHeader.BlockHeight())
	}

	prevBlockHashPtr := input.ResultsBlock.Header.PrevBlockHashPtr()
	if !bytes.Equal(input.PrevBlockHash, prevBlockHashPtr) {
		return errors.New("incorrect previous results block hash")
	}

	if checkedHeader.Timestamp() != input.TransactionsBlock.Header.Timestamp() {
		return fmt.Errorf("mismatching timestamps: txBlock=%v rxBlock=%v", checkedHeader.Timestamp(), input.TransactionsBlock.Header.Timestamp())
	}

	// Check the receipts merkle root matches the receipts.
	receipts := input.ResultsBlock.TransactionReceipts
	calculatedReceiptsRoot, err := calculateReceiptsMerkleRoot(receipts)
	if err != nil {
		fmt.Errorf("error in calculatedReceiptsRoot  blockheight=%v", input.BlockHeight)
		return err
	}
	if !bytes.Equal(checkedHeader.ReceiptsMerkleRootHash(), calculatedReceiptsRoot) {
		fmt.Println("ValidateResultsBlock122 ", calculatedReceiptsRoot, checkedHeader)
		return errors.New("incorrect receipts root hash")
	}

	// Check the hash of the state diff in the block.
	// TODO Statediff not impl - pending https://tree.taiga.io/project/orbs-network/us/535

	// Check hash pointer to the Transactions block of the same height.
	if checkedHeader.BlockHeight() != input.TransactionsBlock.Header.BlockHeight() {
		return fmt.Errorf("mismatching block height: txBlock=%v rxBlock=%v", checkedHeader.BlockHeight(), input.TransactionsBlock.Header.BlockHeight())
	}

	// Check merkle root of the state prior to the block execution, retrieved by calling `StateStorage.GetStateHash`. blockHeight-1
	calculatedPreExecutionStateRootHash, err := getStateHash(ctx, &services.GetStateHashInput{
		BlockHeight: checkedHeader.BlockHeight() - 1,
	})
	if err != nil {
		return err
	}

	if !bytes.Equal(checkedHeader.PreExecutionStateMerkleRootHash(), calculatedPreExecutionStateRootHash.StateMerkleRootHash) {
		return fmt.Errorf("mismatching PreExecutionStateRootHash: expected %v but results block hash %v",
			calculatedPreExecutionStateRootHash, checkedHeader.PreExecutionStateMerkleRootHash())
	}

	// Check transaction id bloom filter (see block format for structure).
	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118

	// Check transaction timestamp bloom filter (see block format for structure).
	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118

	// Validate transaction execution

	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet
	// (creating receipts and state diff). Using the provided header timestamp as a reference timestamp.
	_, err = processTransactionSet(ctx, &services.ProcessTransactionSetInput{
		BlockHeight:        checkedHeader.BlockHeight(),
		SignedTransactions: input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return err
	}

	// Compare the receipts merkle root hash to the one in the block

	// Compare the state diff hash to the one in the block (supports only deterministic execution).

	// TODO https://tree.taiga.io/project/orbs-network/us/535 How to calculate receipts merkle hash root and state diff hash
	// See https://github.com/orbs-network/orbs-spec/issues/111
	//blockMerkleRootHash := checkedHeader.ReceiptsRootHash()

	return nil

}
