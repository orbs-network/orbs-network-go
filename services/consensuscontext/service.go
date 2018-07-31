package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type service struct {
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	stateStorage    services.StateStorage
}

func NewConsensusContext(
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	stateStorage services.StateStorage,
) services.ConsensusContext {

	return &service{
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		stateStorage:    stateStorage,
	}
}

func (s *service) RequestNewTransactionsBlock(input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})
	if err != nil {
		return nil, err
	}

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:       blockstorage.ProtocolVersion,
			BlockHeight:           input.BlockHeight,
			NumSignedTransactions: uint32(len(proposedTransactions.SignedTransactions)),
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         nil,
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil

}

func (s *service) RequestNewResultsBlock(input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsBlock(input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateResultsBlock(input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestOrderingCommittee(input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}

func (s *service) RequestValidationCommittee(input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	panic("Not implemented")
}
