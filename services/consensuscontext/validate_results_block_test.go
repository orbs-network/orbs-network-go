// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testValidators "github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
	stateDiffHash, _ := digest.CalcStateDiffHash(block.ResultsBlock.ContractStateDiffs)
	preExecutionRootHash := &services.GetStateHashOutput{
		StateMerkleRootHash: empty32ByteHash,
	}

	block.ResultsBlock.Header.MutateTransactionsBlockHashPtr(txBlockHashPtr)
	block.ResultsBlock.Header.MutateReceiptsMerkleRootHash(receiptMerkleRoot)
	block.ResultsBlock.Header.MutateStateDiffHash(stateDiffHash)
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

func mockGetStateHashThatReturns(stateRootHash primitives.Sha256, err error) func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
	return func(ctx context.Context, input *services.GetStateHashInput) (*services.GetStateHashOutput, error) {
		return &services.GetStateHashOutput{
			StateMerkleRootHash: stateRootHash,
		}, err
	}
}

func TestResultsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
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
		someRandomHash := hash.CalcSha256([]byte{2})
		err := validateRxTxBlockPtrMatchesActualTxBlock(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutateTransactionsBlockHashPtr(someRandomHash); err != nil {
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
		someRandomHash := hash.CalcSha256([]byte{2})
		err := validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Nil(t, err)
		if err := vcrx.input.ResultsBlock.Header.MutatePrevBlockHashPtr(someRandomHash); err != nil {
			t.Error(err)
		}
		err = validateRxPrevBlockHashPtr(context.Background(), vcrx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error when state's pre-execution merkle root is different between the results block and state storage", func(t *testing.T) {
		vcrx := toRxValidatorContext(cfg)
		manualPreExecutionStateMerkleRootHash1 := hash.CalcSha256([]byte{1})
		manualPreExecutionStateMerkleRootHash2 := hash.CalcSha256([]byte{2})

		// success case - setup the results block and GetStateHash() to return same hash
		successfulGetStateHash := mockGetStateHashThatReturns(manualPreExecutionStateMerkleRootHash1, nil)
		if err := vcrx.input.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(manualPreExecutionStateMerkleRootHash1); err != nil {
			t.Error(err)
		}
		vcrx.getStateHash = successfulGetStateHash
		err := validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Nil(t, err, "results block holds the same pre-execution merkle root that is returned from state storage")

		// GetStateHash returns error
		errorGetStateHash := mockGetStateHashThatReturns(vcrx.input.ResultsBlock.Header.PreExecutionStateMerkleRootHash(), errors.New("Some error"))
		vcrx.getStateHash = errorGetStateHash
		err = validatePreExecutionStateMerkleRoot(context.Background(), vcrx)
		require.Equal(t, ErrGetStateHash, errors.Cause(err), "validation should fail if failed to read the pre-execution merkle root from state storage", err)

		// GetStateHash returns successfully but a mismatching hash
		vcrx.getStateHash = successfulGetStateHash
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

		manualStateDiffHash1 := hash.CalcSha256([]byte{10})
		manualStateDiffHash2 := hash.CalcSha256([]byte{20})

		// Set expected values in results block (they will match those returned from successfulCalculateReceiptsMerkleRoot and successfulCalculateStateDiffHash
		if err := vcrx.input.ResultsBlock.Header.MutateReceiptsMerkleRootHash(manualReceiptsMerkleRoot1); err != nil {
			t.Error(err)
		}
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffHash1); err != nil {
			t.Error(err)
		}

		successfulProcessTransactionSet := MockProcessTransactionSetThatReturns(nil)
		successfulCalcReceiptsMerkleRoot := testValidators.MockCalcReceiptsMerkleRootThatReturns(manualReceiptsMerkleRoot1, nil)
		successfulCalcStateDiffHash := testValidators.MockCalcStateDiffHashThatReturns(manualStateDiffHash1, nil)
		errorProcessTransactionSet := MockProcessTransactionSetThatReturns(errors.New("Some error"))
		errorCalcReceiptsMerkleRoot := testValidators.MockCalcReceiptsMerkleRootThatReturns(nil, errors.New("Some error"))
		errorCalcStateDiffHash := testValidators.MockCalcStateDiffHashThatReturns(nil, errors.New("Some error"))

		// ProcessTransactionSet returns an error - returns ErrProcessTransactionSet
		vcrx.processTransactionSet = errorProcessTransactionSet
		err := validateExecution(context.Background(), vcrx)
		require.Equal(t, ErrProcessTransactionSet, errors.Cause(err), "validation should fail if failed to execute transaction set", err)

		// CalcReceiptsMerkleRoot returns error
		vcrx.processTransactionSet = successfulProcessTransactionSet
		vcrx.calcReceiptsMerkleRoot = errorCalcReceiptsMerkleRoot
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrCalcReceiptsMerkleRoot, errors.Cause(err), "validation should fail if failed to calculate receipts merkle root", err)

		// CalcStateDiffHash returns error
		vcrx.calcReceiptsMerkleRoot = successfulCalcReceiptsMerkleRoot
		vcrx.calcStateDiffHash = errorCalcStateDiffHash
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrCalcStateDiffHash, errors.Cause(err), "validation should fail if failed to calculate state diff merkle root", err)

		// Test the only case where everything is fine - collaborators don't return errors, and there are no mismatches
		vcrx.calcStateDiffHash = successfulCalcStateDiffHash
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
		if err := vcrx.input.ResultsBlock.Header.MutateStateDiffHash(manualStateDiffHash2); err != nil {
			t.Error(err)
		}
		err = validateExecution(context.Background(), vcrx)
		require.Equal(t, validators.ErrMismatchedStateDiffHash, errors.Cause(err), "validation should fail on incorrect post-execution state diff hash", err)
		require.Contains(t, err.Error(), `"BenchmarkToken/616d6f756e74":"e: 0a <==> c: NA"`, "expected error message to include a digest of the state diff comparison")
	})

}

func TestCompare(t *testing.T) {

	expectedDiffs := []*protocol.ContractStateDiff{
		builders.ContractStateDiff().WithContractName("m1").WithStringRecord("mr1", "mv1").WithStringRecord("mrSame", "mvSame").Build(),
		builders.ContractStateDiff().WithContractName("m2").WithStringRecord("mr2", "mv2").Build(),
		builders.ContractStateDiff().WithContractName("m4").WithStringRecord("mr4", "mv4").Build(),
	}

	calculatedDiffs := []*protocol.ContractStateDiff{
		builders.ContractStateDiff().WithContractName("m1").WithStringRecord("mrSame", "mvSame").Build(),
		builders.ContractStateDiff().WithContractName("m4").WithStringRecord("mr4", "mv4").WithStringRecord("mrNew", "mvNew").Build(),
		builders.ContractStateDiff().WithContractName("m2").WithStringRecord("mr2", "mv3").Build(),
		builders.ContractStateDiff().WithContractName("m3").WithStringRecord("mr3", "mv4").Build(),
	}

	require.EqualValues(t, compare(expectedDiffs, calculatedDiffs), map[string]string{"m3/6d7233": "e: NA <==> c: 6d7634", "m1/6d7231": "e: 6d7631 <==> c: NA", "m4/6d724e6577": "e: NA <==> c: 6d764e6577", "m2/6d7232": "e: 6d7632 <==> c: 6d7633"})
}

func MockProcessTransactionSetThatReturns(err error) func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
	someEmptyTxSetThatWeReturnOnlyToPreventErrors := &services.ProcessTransactionSetOutput{
		TransactionReceipts: nil,
		ContractStateDiffs:  nil,
	}
	return func(ctx context.Context, input *services.ProcessTransactionSetInput) (*services.ProcessTransactionSetOutput, error) {
		return someEmptyTxSetThatWeReturnOnlyToPreventErrors, err
	}
}
