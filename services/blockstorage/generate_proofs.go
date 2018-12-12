package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) GenerateReceiptProof(ctx context.Context, input *services.GenerateReceiptProofInput) (*services.GenerateReceiptProofOutput, error) {
	block, err := s.persistence.GetResultsBlock(input.BlockHeight)
	if err != nil {
		return nil, err
	}

	for i, txr := range block.TransactionReceipts {
		if txr.Txhash().Equal(input.Txhash) {

			proof, err := generateProof(block.TransactionReceipts, i)
			if err != nil {
				return nil, err
			}

			// TODO (issue 67) need raw copy
			result := &services.GenerateReceiptProofOutput{
				Proof: (&protocol.ReceiptProofBuilder{
					Header: &protocol.ResultsBlockHeaderBuilder{
						ProtocolVersion:             block.Header.ProtocolVersion(),
						VirtualChainId:              block.Header.VirtualChainId(),
						BlockHeight:                 0,
						PrevBlockHashPtr:            nil,
						Timestamp:                   0,
						ReceiptsRootHash:            block.Header.ReceiptsRootHash(),
						StateDiffHash:               nil,
						TransactionsBlockHashPtr:    nil,
						PreExecutionStateRootHash:   nil,
						TransactionsBloomFilterHash: nil,
						NumTransactionReceipts:      0,
						NumContractStateDiffs:       0,
					},
					BlockProof: &protocol.ResultsBlockProofBuilder{
						TransactionsBlockHash: block.BlockProof.TransactionsBlockHash(),
						Type:                  0,
						BenchmarkConsensus:    nil,
						LeanHelix:             nil,
					},
					ReceiptProof: proof,
					ReceiptIndex: nil, /* i */
					Receipt: &protocol.TransactionReceiptBuilder{
						Txhash:              txr.Txhash(),
						ExecutionResult:     txr.ExecutionResult(),
						OutputArgumentArray: txr.OutputArgumentArray(),
						OutputEventsArray:   txr.OutputEventsArray(),
					},
				}).Build(),
			}
			return result, nil
		}
	}

	return nil, errors.Errorf("could not find transaction inside block %x", input.Txhash)

}

func generateProof(receipts []*protocol.TransactionReceipt, index int) (primitives.MerkleTreeProof, error) {
	rptHashValues := make([]primitives.Sha256, len(receipts))
	for i := 0; i < len(receipts); i++ {
		rptHashValues[i] = digest.CalcReceiptHash(receipts[i])
	}
	proof, err := merkle.NewOrderedTree(rptHashValues).GetProof(index)
	if err != nil {
		return nil, err
	}

	arr := make([]byte, 0, len(proof)*len(proof[0])) // TODO (issue 121) need const for sha
	for _, v := range proof {
		arr = append(arr, v...)
	}

	return primitives.MerkleTreeProof(arr), nil
}
