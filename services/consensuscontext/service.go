package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type Config interface {
	MinimumTransactionsInBlock() int
	BelowMinimalBlockDelayMillis() uint32
}

type service struct {
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	stateStorage    services.StateStorage
	config          Config
}

func NewConsensusContext(
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	stateStorage services.StateStorage,
	config Config,
) services.ConsensusContext {

	return &service{
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		stateStorage:    stateStorage,
		config:          config,
	}
}

func (s *service) RequestNewTransactionsBlock(input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
	txBlock, err := s.createTransactionsBlock(input.BlockHeight, input.PrevBlockHash)

	if err != nil {
		return nil, err
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) RequestNewResultsBlock(input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {

	rxBlock := s.createResultsBlock(input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)

	return &services.RequestNewResultsBlockOutput{
		ResultsBlock: rxBlock,
	}, nil
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
