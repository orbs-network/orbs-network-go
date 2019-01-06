package validators

import (
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
