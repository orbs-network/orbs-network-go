package consensuscontext

import (
	"bytes"
	"context"
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

var ErrMismatchedTxRxBlockHeight = errors.New("mismatched block height between transactions and results")
var ErrMismatchedTxRxTimestamps = errors.New("mismatched timestamp between transactions and results")
var ErrMismatchedTxHashPtrToActualTxBlock = errors.New("mismatched tx block hash ptr to actual tx block hash")
var ErrMismatchedReceiptsRootHash = errors.New("receipt merkleRoot is different between results block header and calculated transaction receipts")
var ErrMismatchedStateDiffHash = errors.New("state diff merkleRoot is different between results block header and calculated transaction receipts")
var ErrGetStateHash = errors.New("failed in GetStateHash() so cannot retrieve pre-execution state diff merkleRoot from previous block")
var ErrMismatchedPreExecutionStateMerkleRoot = errors.New("pre-execution state diff merkleRoot is different between results block header and extracted from state storage for previous block")
var ErrProcessTransactionSet = errors.New("failed in ProcessTransactionSet()")
var ErrCalculateReceiptsMerkleRoot = errors.New("failed in CalculateReceiptsMerkleRoot()")
var ErrCalculateStateDiffMerkleRoot = errors.New("failed in ErrCalculateStateDiffMerkleRoot()")

type txValidator func(ctx context.Context, vctx *txValidatorContext) error

type txValidatorContext struct {
	protocolVersion        primitives.ProtocolVersion
	virtualChainId         primitives.VirtualChainId
	allowedTimestampJitter time.Duration
	input                  *services.ValidateTransactionsBlockInput
	txOrderValidator       func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error)
}

func validateTxProtocolVersion(ctx context.Context, vctx *txValidatorContext) error {
	expectedProtocolVersion := vctx.protocolVersion
	checkedProtocolVersion := vctx.input.TransactionsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}
	return nil
}

func validateTxVirtualChainID(ctx context.Context, vctx *txValidatorContext) error {
	expectedVirtualChainId := vctx.virtualChainId
	checkedVirtualChainId := vctx.input.TransactionsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != vctx.virtualChainId {
		return errors.Wrapf(ErrMismatchedVirtualChainID, "expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}
	return nil
}

func validateTxBlockHeight(ctx context.Context, vctx *txValidatorContext) error {
	checkedBlockHeight := vctx.input.TransactionsBlock.Header.BlockHeight()
	expectedBlockHeight := vctx.input.BlockHeight
	if checkedBlockHeight != expectedBlockHeight {
		return ErrMismatchedBlockHeight
	}
	return nil
}

func validateTxPrevBlockHashPtr(ctx context.Context, vctx *txValidatorContext) error {
	expectedPrevBlockHashPtr := vctx.input.PrevBlockHash
	prevBlockHashPtr := vctx.input.TransactionsBlock.Header.PrevBlockHashPtr()
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedPrevBlockHash, "expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}
	return nil
}

func validateTxTransactionsBlockTimestamp(ctx context.Context, vctx *txValidatorContext) error {
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

func validateTxTransactionsBlockMerkleRoot(ctx context.Context, vctx *txValidatorContext) error {
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	txMerkleRoot := vctx.input.TransactionsBlock.Header.TransactionsRootHash()
	if expectedTxMerkleRoot, err := calculateTransactionsMerkleRoot(vctx.input.TransactionsBlock.SignedTransactions); err != nil {
		return err
	} else if !bytes.Equal(txMerkleRoot, expectedTxMerkleRoot) {
		return errors.Wrapf(ErrMismatchedTxMerkleRoot, "expected %v actual %v", expectedTxMerkleRoot, txMerkleRoot)
	}
	return nil
}

func validateTxMetadataHash(ctx context.Context, vctx *txValidatorContext) error {
	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(vctx.input.TransactionsBlock.Metadata)
	metadataHash := vctx.input.TransactionsBlock.Header.MetadataHash()
	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
		return errors.Wrapf(ErrMismatchedMetadataHash, "expected %v actual %v", expectedMetaDataHash, metadataHash)
	}
	return nil
}

func validateTxTransactionOrdering(ctx context.Context, vctx *txValidatorContext) error {
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

	vctx := &txValidatorContext{
		protocolVersion:        s.config.ProtocolVersion(),
		virtualChainId:         s.config.VirtualChainId(),
		allowedTimestampJitter: s.config.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
		txOrderValidator:       s.transactionPool.ValidateTransactionsForOrdering,
	}

	validators := []txValidator{
		validateTxProtocolVersion,
		validateTxVirtualChainID,
		validateTxBlockHeight,
		validateTxPrevBlockHashPtr,
		validateTxTransactionsBlockTimestamp,
		validateTxTransactionsBlockMerkleRoot,
		validateTxMetadataHash,
		validateTxTransactionOrdering,
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
