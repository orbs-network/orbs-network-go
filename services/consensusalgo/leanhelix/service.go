package leanhelix

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
)

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodeId() string
}

type service struct {
	gossip               gossiptopics.LeanHelix
	blockStorage         services.BlockStorage
	transactionPool      services.TransactionPool
	consensusContext     services.ConsensusContext
	events               instrumentation.Reporting
	loopControl          instrumentation.LoopControl
	votesForCurrentRound chan bool
	config               Config
	preparedBlock        *protocol.BlockPairContainer
	commitCond           *sync.Cond
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
		events:           events,
		loopControl:      loopControl,
		config:           config,
		commitCond:       sync.NewCond(&sync.Mutex{}),
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
	s.preparedBlock = input.Message.BlockPair // each node will save this block
	println("block after preprepare", s.preparedBlock.TransactionsBlock.Header.String())
	return s.gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
}

func (s *service) HandleLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	// currently only leader should handle prepare
	if s.votesForCurrentRound != nil {
		s.events.Info(fmt.Sprintf("received vote"))
		s.votesForCurrentRound <- true
	}
	return nil, nil
}

func (s *service) HandleLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: s.preparedBlock,
	})
	s.preparedBlock = nil
	s.commitCond.Signal()
	return nil, nil
}

func (s *service) HandleLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) HandleLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) buildNextBlock(transaction *protocol.SignedTransaction) bool {
	votes, err := s.requestConsensusFor(transaction)
	if err != nil {
		s.events.Info(instrumentation.ConsensusError)
		return false
	}
	gotConsensus := true
	// asking for 2/3 or the votes because, strangely enough, we fail to vote for ourselves
	for i := uint32(0); i < s.config.NetworkSize(0); i++ {
		gotConsensus = gotConsensus && <-votes
	}

	// FIXME: related to gossip
	// close(s.votesForCurrentRound)

	if gotConsensus {
		if s.preparedBlock == nil {
			panic(fmt.Sprintf("Node [%s] is trying to commit a block that wasn't prepared", s.config.NodeId()))
		}
		s.gossip.SendLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
	}
	s.commitCond.Wait()
	close(s.votesForCurrentRound)
	return gotConsensus
}

func (s *service) buildBlocksEventLoop() {
	var currentBlock *protocol.SignedTransaction
	s.commitCond.L.Lock()
	s.loopControl.NewLoop("consensus_round", func() {
		if currentBlock == nil {
			res, _ := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
				MaxNumberOfTransactions: 1,
			})
			currentBlock = res.SignedTransactions[0]
		}
		if s.buildNextBlock(currentBlock) {
			currentBlock = nil
		}
	})
}

func (s *service) requestConsensusFor(transaction *protocol.SignedTransaction) (chan bool, error) {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion: blockstorage.ProtocolVersion,
			}).Build(),
			SignedTransactions: []*protocol.SignedTransaction{transaction},
		},
	}

	println("block before preprepare", blockPair.TransactionsBlock.Header.String())
	message := &gossipmessages.LeanHelixPrePrepareMessage{
		BlockPair: blockPair,
	}
	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: message,
	}) //TODO send the actual input, not just a single tx bytes
	if err == nil {
		s.votesForCurrentRound = make(chan bool)
	} else {
		s.votesForCurrentRound = nil
	}
	return s.votesForCurrentRound, err
}
