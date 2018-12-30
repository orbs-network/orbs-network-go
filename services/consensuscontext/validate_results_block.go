package consensuscontext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

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
	if !bytes.Equal(checkedHeader.ReceiptsRootHash(), calculatedReceiptsRoot) {
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

	if !bytes.Equal(checkedHeader.PreExecutionStateRootHash(), calculatedPreExecutionStateRootHash.StateRootHash) {
		return fmt.Errorf("mismatching PreExecutionStateRootHash: expected %v but results block hash %v",
			calculatedPreExecutionStateRootHash, checkedHeader.PreExecutionStateRootHash())
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
