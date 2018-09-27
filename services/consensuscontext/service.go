package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type Config interface {
	ConsensusContextMaximumTransactionsInBlock() uint32
	ConsensusContextMinimumTransactionsInBlock() uint32
	ConsensusContextMinimalBlockDelay() time.Duration
}

type service struct {
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	stateStorage    services.StateStorage
	config          Config
	reporting       log.BasicLogger
}

func NewConsensusContext(
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	stateStorage services.StateStorage,
	config Config,
	reporting log.BasicLogger,
) services.ConsensusContext {

	return &service{
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		stateStorage:    stateStorage,
		config:          config,
		reporting:       reporting.For(log.Service("consensus-context")),
	}
}

func (s *service) RequestNewTransactionsBlock(input *services.RequestNewTransactionsBlockInput) (*services.RequestNewTransactionsBlockOutput, error) {
	txBlock, err := s.createTransactionsBlock(input.BlockHeight, input.PrevBlockHash)
	if err != nil {
		return nil, err
	}

	s.reporting.Info("created Transactions block", log.Int("num-transactions", len(txBlock.SignedTransactions)), log.Stringable("transactions-block", txBlock))

	for _, tx := range txBlock.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		s.reporting.Info("transaction entered transactions block", log.String("flow", "checkpoint"), log.Stringable("txHash", txHash), log.BlockHeight(txBlock.Header.BlockHeight()))
	}

	return &services.RequestNewTransactionsBlockOutput{
		TransactionsBlock: txBlock,
	}, nil
}

func (s *service) RequestNewResultsBlock(input *services.RequestNewResultsBlockInput) (*services.RequestNewResultsBlockOutput, error) {
	rxBlock, err := s.createResultsBlock(input.BlockHeight, input.PrevBlockHash, input.TransactionsBlock)
	if err != nil {
		return nil, err
	}

	s.reporting.Info("created Results block", log.Stringable("results-block", rxBlock))

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
