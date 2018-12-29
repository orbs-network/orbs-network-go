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

// TODO Write tests in validate_block_test.go

func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (
	*services.ValidateTransactionsBlockOutput, error) {
	err := ValidateTransactionsBlockInternal(ctx, input,
		s.config.ProtocolVersion(), s.config.VirtualChainId(), s.config.ConsensusContextSystemTimestampAllowedJitter(),
		s.transactionPool.ValidateTransactionsForOrdering)
	return &services.ValidateTransactionsBlockOutput{}, err
}

func ValidateTransactionsBlockInternal(
	ctx context.Context,
	input *services.ValidateTransactionsBlockInput,
	expectedProtocolVersion primitives.ProtocolVersion,
	expectedVirtualChainId primitives.VirtualChainId,
	allowedTimestampJitter time.Duration,
	validateTransactionsForOrdering func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (
		*services.ValidateTransactionsForOrderingOutput, error)) error {

	checkedBlockHeight := input.TransactionsBlock.Header.BlockHeight()
	expectedBlockHeight := input.BlockHeight
	if checkedBlockHeight != expectedBlockHeight {
		return fmt.Errorf("ValidateTransactionsBlock mismatching blockHeight: expected %v actual %v", expectedBlockHeight, checkedBlockHeight)
	}

	checkedProtocolVersion := input.TransactionsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return fmt.Errorf("ValidateTransactionsBlock incorrect protocol version: expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}

	checkedVirtualChainId := input.TransactionsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != expectedVirtualChainId {
		return fmt.Errorf("ValidateTransactionsBlock incorrect virtualChainId: expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}

	prevBlockHashPtr := input.TransactionsBlock.Header.PrevBlockHashPtr()
	expectedPrevBlockHashPtr := input.PrevBlockHash
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return fmt.Errorf("ValidateTransactionsBlock mismatching previous block pointer: expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}

	prevBlockTimestamp := input.PrevBlockTimestamp
	currentBlockTimestamp := input.TransactionsBlock.Header.Timestamp()
	now := time.Now()
	if !isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter) {
		return fmt.Errorf("ValidateTransactionsBlock current block timestamp is invalid. currentTimestamp %v prevTimestamp %v now %v allowed jitter %v",
			currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter)
	}

	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	txMerkleRoot := input.TransactionsBlock.Header.TransactionsRootHash()
	if expectedTxMerkleRoot, err := calculateTransactionsMerkleRoot(input.TransactionsBlock.SignedTransactions); err != nil {
		return errors.Wrapf(err, "ValidateTransactionsBlock error calculateTransactionsMerkleRoot")
	} else if !bytes.Equal(txMerkleRoot, expectedTxMerkleRoot) {
		return fmt.Errorf("ValidateTransactionsBlock mismatching transaction merkleRoot: expected %v actual %v", expectedTxMerkleRoot, txMerkleRoot)
	}

	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	metadataHash := input.TransactionsBlock.Header.MetadataHash()
	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(input.TransactionsBlock.Metadata)
	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
		return fmt.Errorf("ValidateTransactionsBlock mismatching transaction metadataHash: expected %v actual %v", expectedMetaDataHash, metadataHash)
	}

	// TODO v1 "Check timestamp is within configurable allowed jitter of system timestamp, and later than previous block"

	validationInput := &services.ValidateTransactionsForOrderingInput{
		BlockHeight:        input.TransactionsBlock.Header.BlockHeight(),
		BlockTimestamp:     input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions: input.TransactionsBlock.SignedTransactions,
	}

	_, err := validateTransactionsForOrdering(ctx, validationInput)
	if err != nil {
		return err
	}
	return nil

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
	fmt.Println("ValidateResultsBlock1 ")
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
	if !bytes.Equal(checkedHeader.ReceiptsRootHash(), calculatedReceiptsRoot) {
		fmt.Println("ValidateResultsBlock122 ", calculatedReceiptsRoot, checkedHeader)
		return errors.New("incorrect receipts root hash")
	}
	fmt.Println("ValidateResultsBlock2 ", checkedHeader.BlockHeight())
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
	fmt.Println("ValidateResultsBlock3 ", checkedHeader.BlockHeight())
	if !bytes.Equal(checkedHeader.PreExecutionStateRootHash(), calculatedPreExecutionStateRootHash.StateRootHash) {
		return fmt.Errorf("mismatching PreExecutionStateRootHash: expected %v but results block hash %v",
			calculatedPreExecutionStateRootHash, checkedHeader.PreExecutionStateRootHash())
	}
	fmt.Println("ValidateResultsBlock4 ", checkedHeader.BlockHeight())

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

// OLD WORKING CODE

// Validates another node's proposed block.
// Performed upon request from consensus algo when receiving a proposal during a live consensus round
//func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
//
//	checkedHeader := input.TransactionsBlock.Header
//	expectedProtocolVersion := s.config.ProtocolVersion()
//	expectedVirtualChainId := s.config.VirtualChainId()
//
//	txs := input.TransactionsBlock.SignedTransactions
//	txMerkleRootHash := checkedHeader.TransactionsRootHash()
//
//	prevBlockHashPtr := checkedHeader.PrevBlockHashPtr()
//
//	blockProtocolVersion := checkedHeader.ProtocolVersion()
//	blockVirtualChainId := checkedHeader.VirtualChainId()
//
//	if blockProtocolVersion != expectedProtocolVersion {
//		return nil, fmt.Errorf("incorrect protocol version: expected %v but block has %v", expectedProtocolVersion, blockProtocolVersion)
//	}
//	if blockVirtualChainId != expectedVirtualChainId {
//		return nil, fmt.Errorf("incorrect virtual chain ID: expected %v but block has %v", expectedVirtualChainId, blockVirtualChainId)
//	}
//	if input.BlockHeight != checkedHeader.BlockHeight() {
//		return nil, fmt.Errorf("mismatching blockHeight: input %v checkedHeader %v", input.BlockHeight, checkedHeader.BlockHeight())
//	}
//	calculatedTxRoot, err := calculateTransactionsRootHash(txs)
//	if err != nil {
//		return nil, err
//	}
//	if !bytes.Equal(txMerkleRootHash, calculatedTxRoot) {
//		return nil, errors.New("incorrect transactions root hash")
//	}
//	calculatedPrevBlockHashPtr := calculatePrevBlockHashPtr(input.TransactionsBlock)
//	if !bytes.Equal(prevBlockHashPtr, calculatedPrevBlockHashPtr) {
//		return nil, errors.New("incorrect previous block hash")
//	}
//
//	// TODO v1 "Check timestamp is within configurable allowed jitter of system timestamp, and later than previous block"
//
//	// TODO "Check transaction merkle root hash" https://github.com/orbs-network/orbs-spec/issues/118
//
//	// TODO "Check metadata hash" https://tree.taiga.io/project/orbs-network/us/535
//
//	validationInput := &services.ValidateTransactionsForOrderingInput{
//		BlockHeight:        input.BlockHeight,
//		BlockTimestamp:     input.TransactionsBlock.Header.Timestamp(),
//		SignedTransactions: input.TransactionsBlock.SignedTransactions,
//	}
//
//	_, err = s.transactionPool.ValidateTransactionsForOrdering(ctx, validationInput)
//	if err != nil {
//		return nil, err
//	}
//
//	return &services.ValidateTransactionsBlockOutput{}, nil
//
//}
//
//func (s *service) ValidateResultsBlock(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
//	expectedProtocolVersion := s.config.ProtocolVersion()
//	expectedVirtualChainId := s.config.VirtualChainId()
//
//	checkedHeader := input.ResultsBlock.Header
//	blockProtocolVersion := checkedHeader.ProtocolVersion()
//	blockVirtualChainId := checkedHeader.VirtualChainId()
//	if blockProtocolVersion != expectedProtocolVersion {
//		return nil, fmt.Errorf("incorrect protocol version: expected %v but block has %v", expectedProtocolVersion, blockProtocolVersion)
//	}
//	if blockVirtualChainId != expectedVirtualChainId {
//		return nil, fmt.Errorf("incorrect virtual chain ID: expected %v but block has %v", expectedVirtualChainId, blockVirtualChainId)
//	}
//	if input.BlockHeight != checkedHeader.BlockHeight() {
//		return nil, fmt.Errorf("mismatching blockHeight: input %v checkedHeader %v", input.BlockHeight, checkedHeader.BlockHeight())
//	}
//
//	prevBlockHashPtr := input.ResultsBlock.Header.PrevBlockHashPtr()
//	if !bytes.Equal(input.PrevBlockHash, prevBlockHashPtr) {
//		return nil, errors.New("incorrect previous results block hash")
//	}
//	if checkedHeader.Timestamp() != input.TransactionsBlock.Header.Timestamp() {
//		return nil, fmt.Errorf("mismatching timestamps: txBlock=%v rxBlock=%v", checkedHeader.Timestamp(), input.TransactionsBlock.Header.Timestamp())
//	}
//	// Check the receipts merkle root matches the receipts.
//	receipts := input.ResultsBlock.TransactionReceipts
//	calculatedReceiptsRoot, err := calculateReceiptsRootHash(receipts)
//	if err != nil {
//		return nil, err
//	}
//	if !bytes.Equal(checkedHeader.ReceiptsRootHash(), calculatedReceiptsRoot) {
//		return nil, errors.New("incorrect receipts root hash")
//	}
//
//	// Check the hash of the state diff in the block.
//	// TODO Statediff not impl - pending https://tree.taiga.io/project/orbs-network/us/535
//
//	// Check hash pointer to the Transactions block of the same height.
//	if checkedHeader.BlockHeight() != input.TransactionsBlock.Header.BlockHeight() {
//		return nil, fmt.Errorf("mismatching block height: txBlock=%v rxBlock=%v", checkedHeader.BlockHeight(), input.TransactionsBlock.Header.BlockHeight())
//	}
//
//	// Check merkle root of the state prior to the block execution, retrieved by calling `StateStorage.GetStateHash`.
//
//	calculatedPreExecutionStateRootHash, err := s.stateStorage.GetStateHash(ctx, &services.GetStateHashInput{
//		BlockHeight: checkedHeader.BlockHeight(),
//	})
//	if err != nil {
//		return nil, err
//	}
//	if !bytes.Equal(checkedHeader.PreExecutionStateRootHash(), calculatedPreExecutionStateRootHash.StateRootHash) {
//		return nil, fmt.Errorf("mismatching PreExecutionStateRootHash: expected %v but results block hash %v",
//			calculatedPreExecutionStateRootHash, checkedHeader.PreExecutionStateRootHash())
//	}
//
//	// Check transaction id bloom filter (see block format for structure).
//	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118
//
//	// Check transaction timestamp bloom filter (see block format for structure).
//	// TODO Pending spec https://github.com/orbs-network/orbs-spec/issues/118
//
//	// Validate transaction execution
//
//	// Execute the ordered transactions set by calling VirtualMachine.ProcessTransactionSet
//	// (creating receipts and state diff). Using the provided header timestamp as a reference timestamp.
//	_, err = s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
//		BlockHeight:        checkedHeader.BlockHeight(),
//		SignedTransactions: input.TransactionsBlock.SignedTransactions,
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	// Compare the receipts merkle root hash to the one in the block
//
//	// Compare the state diff hash to the one in the block (supports only deterministic execution).
//
//	// TODO https://tree.taiga.io/project/orbs-network/us/535 How to calculate receipts merkle hash root and state diff hash
//	// See https://github.com/orbs-network/orbs-spec/issues/111
//	//blockMerkleRootHash := checkedHeader.ReceiptsRootHash()
//
//	return &services.ValidateResultsBlockOutput{}, nil
//
//}
