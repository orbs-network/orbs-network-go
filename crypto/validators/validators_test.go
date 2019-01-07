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

func TestValidateBlockHash(t *testing.T) {

	tamperedTimestamp := primitives.TimestampNano(time.Now().UnixNano() + 1000)
	tamperedPrevBlockHash := hash.CalcSha256([]byte{9, 9, 9})
	tamperedMetadataHash := hash.CalcSha256([]byte{9, 9, 7})
	tamperedTxMerkleRoot := hash.CalcSha256([]byte{9, 9, 6})
	tamperedHash := hash.CalcSha256([]byte{6, 6, 6})
	var mutations = []struct {
		name          string
		mutate        func(*BlockValidatorContext)
		expectSuccess bool
	}{
		{name: "valid block", mutate: func(c *BlockValidatorContext) {}, expectSuccess: true},
		{name: "nil transaction block", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock = nil }, expectSuccess: false},
		{name: "nil results block", mutate: func(c *BlockValidatorContext) { c.ResultsBlock = nil }, expectSuccess: false},
		{name: "tampered transactions block protocolVersion", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock.Header.MutateProtocolVersion(1234) }, expectSuccess: false},
		{name: "tampered transactions block virtual chain ID", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock.Header.MutateVirtualChainId(3456) }, expectSuccess: false},
		{name: "tampered transactions block height", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock.Header.MutateBlockHeight(999) }, expectSuccess: false},
		{name: "tampered transactions prev block hash", mutate: func(c *BlockValidatorContext) {
			c.TransactionsBlock.Header.MutatePrevBlockHashPtr(tamperedPrevBlockHash)
		}, expectSuccess: false},
		{name: "tampered transactions metadata hash", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock.Header.MutateMetadataHash(tamperedMetadataHash) }, expectSuccess: false},
		{name: "tampered transactions merkle root hash", mutate: func(c *BlockValidatorContext) {
			c.TransactionsBlock.Header.MutateTransactionsMerkleRootHash(tamperedTxMerkleRoot)
		}, expectSuccess: false},
		{name: "tampered transactions block timestamp", mutate: func(c *BlockValidatorContext) { c.TransactionsBlock.Header.MutateTimestamp(tamperedTimestamp) }, expectSuccess: false},
		{name: "tampered results block protocolVersion", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateProtocolVersion(1234) }, expectSuccess: false},
		{name: "tampered results block virtual chain ID", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateVirtualChainId(4567) }, expectSuccess: false},
		{name: "tampered results block height", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateBlockHeight(998) }, expectSuccess: false},
		{name: "tampered results prev block hash", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutatePrevBlockHashPtr(tamperedPrevBlockHash) }, expectSuccess: false},
		{name: "tampered results block timestamp", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateTimestamp(tamperedTimestamp) }, expectSuccess: false},
		{name: "tampered results block transactions block hash ptr", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateTransactionsBlockHashPtr(tamperedHash) }, expectSuccess: false},
		{name: "tampered results block receipts merkle root hash", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateReceiptsMerkleRootHash(tamperedHash) }, expectSuccess: false},
		{name: "tampered results block state diff hash", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateStateDiffHash(tamperedHash) }, expectSuccess: false},
		{name: "tampered results block pre-execution state merkle root hash", mutate: func(c *BlockValidatorContext) {
			c.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(tamperedHash)
		}, expectSuccess: false},
		{name: "tampered results block num transactions receipts", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateNumTransactionReceipts(999) }, expectSuccess: false},
		{name: "tampered results block num contract diffs", mutate: func(c *BlockValidatorContext) { c.ResultsBlock.Header.MutateNumContractStateDiffs(888) }, expectSuccess: false},
	}

	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			blockUnderTest := validBlockValidatorContext()
			m.mutate(blockUnderTest)
			if m.expectSuccess {
				require.Nil(t, ValidateBlockHash(blockUnderTest), m.name)
			} else {
				require.Error(t, ValidateBlockHash(blockUnderTest), m.name)
			}
		})
	}
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
