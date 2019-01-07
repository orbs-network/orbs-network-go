package validators

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestValidateTransactionsBlockMerkleRoot(t *testing.T) {
	txMerkleRoot1 := hash.CalcSha256([]byte{1})
	block := validators.BuildValidTestBlock()
	bvcx := &BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	}
	if err := bvcx.TransactionsBlock.Header.MutateTransactionsMerkleRootHash(txMerkleRoot1); err != nil {
		t.Error(err)
	}

	err := ValidateTransactionsBlockMerkleRoot(bvcx)
	require.Equal(t, ErrMismatchedTxMerkleRoot, errors.Cause(err), "validation should fail on incorrect transaction root hash", err)

}

// TODO(v1) at least think about it - some kind of mutation testing should be done here as every tampered bit of block data should throw an error
// Suggestion: table test: Start with validBlockValidatorContext(), running a mutator() that modifies a single property, then require(error).
func TestValidateBlockHash(t *testing.T) {

	t.Run("should return error on nil transaction or results block", func(t *testing.T) {
		emptyBlock := &BlockValidatorContext{
			TransactionsBlock: nil,
			ResultsBlock:      nil,
		}
		require.Error(t, ValidateBlockHash(emptyBlock), "should return error on nil transaction or results block")
	})

	t.Run("should return error on tampered transactions block", func(t *testing.T) {
		ctxToTest := validBlockValidatorContext()
		ctxToTest.TransactionsBlock.Header.MutateTimestamp(primitives.TimestampNano(time.Now().UnixNano() + 1000))
		require.Error(t, ValidateBlockHash(ctxToTest), "hash validation of tampered transaction block should return error")
	})

	t.Run("should return error on tampered results block", func(t *testing.T) {
		ctxToTest := validBlockValidatorContext()
		ctxToTest.ResultsBlock.Header.MutateTimestamp(primitives.TimestampNano(time.Now().UnixNano() + 1000))
		require.Error(t, ValidateBlockHash(ctxToTest), "hash validation of tampered results block should return error")
	})

	t.Run("should return nil on block with valid hashes", func(t *testing.T) {
		require.Nil(t, ValidateBlockHash(validBlockValidatorContext()), "should return nil on block with valid hashes")
	})
}

func validBlockValidatorContext() *BlockValidatorContext {
	validBlock := validators.BuildValidTestBlock()
	calculatedHashOfValidBlock := []byte(digest.CalcBlockHash(validBlock.TransactionsBlock, validBlock.ResultsBlock))
	return &BlockValidatorContext{
		TransactionsBlock: validBlock.TransactionsBlock,
		ResultsBlock:      validBlock.ResultsBlock,
		ExpectedBlockHash: calculatedHashOfValidBlock,
	}
}
