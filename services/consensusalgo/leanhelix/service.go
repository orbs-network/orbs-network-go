package leanhelix

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodePublicKey() primitives.Ed25519Pkey
}

type service struct {
	gossip                   gossiptopics.LeanHelix
	blockStorage             services.BlockStorage
	transactionPool          services.TransactionPool
	consensusContext         services.ConsensusContext
	reporting                instrumentation.Reporting
	loopControl              instrumentation.LoopControl
	config                   Config
	lastCommittedBlockHeight primitives.BlockHeight
	votesForActiveRound      map[primitives.BlockHeight]chan bool
	blockForActiveRound      map[primitives.BlockHeight]*protocol.BlockPairContainer
}

func NewLeanHelixConsensusAlgo(
	gossip gossiptopics.LeanHelix,
	blockStorage services.BlockStorage,
	transactionPool services.TransactionPool,
	consensusContext services.ConsensusContext,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	config Config,
	isLeader bool,
) services.ConsensusAlgoLeanHelix {

	s := &service{
		gossip:           gossip,
		blockStorage:     blockStorage,
		transactionPool:  transactionPool,
		consensusContext: consensusContext,
		reporting:        events,
		loopControl:      loopControl,
		config:           config,
		lastCommittedBlockHeight: 0, // TODO: improve
		votesForActiveRound:      make(map[primitives.BlockHeight]chan bool),
		blockForActiveRound:      make(map[primitives.BlockHeight]*protocol.BlockPairContainer),
	}

	gossip.RegisterLeanHelixHandler(s)
	if isLeader {
		go s.buildBlocksEventLoop()
	}
	return s
}

func (s *service) OnNewConsensusRound(input *services.OnNewConsensusRoundInput) (*services.OnNewConsensusRoundOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleTransactionsBlock(input *handlers.HandleTransactionsBlockInput) (*handlers.HandleTransactionsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleResultsBlock(input *handlers.HandleResultsBlockInput) (*handlers.HandleResultsBlockOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixPrePrepare(input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	blockHeight := input.Message.BlockPair.TransactionsBlock.Header.BlockHeight()
	s.blockForActiveRound[blockHeight] = input.Message.BlockPair
	return s.gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
}

func (s *service) HandleLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	// currently only leader should handle prepare
	// TODO: we assume we only get votes for the active round, in the real world we can't assume this
	votes, found := s.votesForActiveRound[s.lastCommittedBlockHeight+1]
	if !found {
		panic("received votes without an active round")
	}
	votes <- true
	return nil, nil
}

func (s *service) HandleLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: s.blockForActiveRound[s.lastCommittedBlockHeight+1],
	})
	delete(s.blockForActiveRound, s.lastCommittedBlockHeight+1)
	s.lastCommittedBlockHeight += 1
	return nil, nil
}

func (s *service) HandleLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) buildBlocksEventLoop() {
	var transactionsForActiveRound []*protocol.SignedTransaction
	s.loopControl.NewLoop("consensus_round", func() {
		if transactionsForActiveRound == nil {
			res, _ := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
				MaxNumberOfTransactions: 1,
			})
			transactionsForActiveRound = res.SignedTransactions
		}
		if s.buildNextBlock(transactionsForActiveRound) {
			transactionsForActiveRound = nil
		}
	})
}

// returns true if the block was committed successfully and we can move to the next block
func (s *service) buildNextBlock(transactionsForBlock []*protocol.SignedTransaction) bool {
	err := s.requestConsensusFor(transactionsForBlock)
	if err != nil {
		s.reporting.Info(instrumentation.ConsensusError)
		return false
	}

	gotConsensus := true
	// asking for votes from everybody except ourselves
	for i := uint32(0); i < s.config.NetworkSize(0)-1; i++ {
		gotConsensus = gotConsensus && <-s.votesForActiveRound[s.lastCommittedBlockHeight+1]
	}

	if gotConsensus {

		blockPair, found := s.blockForActiveRound[s.lastCommittedBlockHeight+1]
		if !found {
			panic(fmt.Sprintf("Node [%v] is trying to commit a block that wasn't prepared", s.config.NodePublicKey()))
		}

		//fmt.Printf("\nTK LEADER COMMIT: %v\n", s.preparedBlock.TransactionsBlock.SignedTransactions[0].Transaction().InputArgumentsIterator().NextInputArguments().Uint64Value())
		s.blockStorage.CommitBlock(&services.CommitBlockInput{
			BlockPair: blockPair,
		})
		s.gossip.SendLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
		close(s.votesForActiveRound[s.lastCommittedBlockHeight+1])
		delete(s.votesForActiveRound, s.lastCommittedBlockHeight+1)
		delete(s.blockForActiveRound, s.lastCommittedBlockHeight+1)
		s.lastCommittedBlockHeight += 1
	}

	return gotConsensus
}

func (s *service) requestConsensusFor(transactionsForBlock []*protocol.SignedTransaction) error {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion: blockstorage.ProtocolVersion,
				BlockHeight:     primitives.BlockHeight(s.lastCommittedBlockHeight + 1),
			}).Build(),
			SignedTransactions: transactionsForBlock,
		},
	}

	s.votesForActiveRound[s.lastCommittedBlockHeight+1] = make(chan bool)
	s.blockForActiveRound[s.lastCommittedBlockHeight+1] = blockPair
	message := &gossipmessages.LeanHelixPrePrepareMessage{
		BlockPair: blockPair,
	}

	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: message,
	})

	return err
}
