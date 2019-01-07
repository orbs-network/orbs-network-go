package validators

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
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

func TestValidateBlockHash(t *testing.T) {

	t.Run("should return error on nil transaction or results block", func(t *testing.T) {
		emptyBlock := &BlockValidatorContext{
			TransactionsBlock: nil,
			ResultsBlock:      nil,
		}
		require.Error(t, ValidateBlockHash(emptyBlock), "should return error on nil transaction or results block")
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
