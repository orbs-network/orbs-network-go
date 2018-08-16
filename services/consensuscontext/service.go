package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
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
	reporting       instrumentation.BasicLogger
}

func NewConsensusContext(
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	stateStorage services.StateStorage,
	config Config,
	reporting instrumentation.BasicLogger,
) services.ConsensusContext {

	return &service{
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		stateStorage:    stateStorage,
		config:          config,
		reporting:       reporting.For(instrumentation.Service("consensus-context")),
	}
}

func (s *service) RequestNewTransactionsBlock(input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
	txBlock, err := s.createTransactionsBlock(input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	s.reporting.Info("created Transactions block", instrumentation.Int("num-transactions", len(txBlock.SignedTransactions)), instrumentation.Stringable("transactions-block", txBlock))

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) RequestNewResultsBlock(input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	rxBlock, err := s.createResultsBlock(input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)
	if err != nil {
		return nil, err
	}

	s.reporting.Info("created Results block", instrumentation.Stringable("results-block", rxBlock))

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
