package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) createTransactionsBlock(blockHeight primitives.BlockHeight) (*protocol.TransactionsBlockContainer, error) {

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})

	if err != nil {
		return nil, err
	}

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:       blockstorage.ProtocolVersion,
			BlockHeight:           blockHeight,
			NumSignedTransactions: uint32(len(proposedTransactions.SignedTransactions)),
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         nil,
	}

	return txBlock, nil
}

func (s *service) createResultsBlock(blockHeight primitives.BlockHeight) *protocol.ResultsBlockContainer {
	rxBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			ProtocolVersion: blockstorage.ProtocolVersion,
			BlockHeight:     blockHeight,
		}).Build(),
		TransactionReceipts: nil,
		ContractStateDiffs:  nil,
		BlockProof:          nil,
	}

	return rxBlock
}
