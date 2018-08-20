package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func (s *service) CommitTransactionReceipts(input *services.CommitTransactionReceiptsInput) (*services.CommitTransactionReceiptsOutput, error) {
	if input.LastCommittedBlockHeight != s.lastCommittedBlockHeight+1 {
		return &services.CommitTransactionReceiptsOutput{
			NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
			LastCommittedBlockHeight: s.lastCommittedBlockHeight,
		}, nil
	}

	var myReceipts []*protocol.TransactionReceipt

	for _, receipt := range input.TransactionReceipts {
		s.committedPool.add(receipt)
		if removedTx := s.pendingPool.remove(receipt.Txhash()); s.originatedFromMyPublicApi(removedTx) {
			myReceipts = append(myReceipts, receipt)
		}
	}

	s.lastCommittedBlockHeight = input.ResultsBlockHeader.BlockHeight()
	s.lastCommittedBlockTimestamp = input.ResultsBlockHeader.Timestamp()

	s.blockTracker.IncrementHeight()

	for _, handler := range s.transactionResultsHandlers {
		handler.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
			BlockHeight:         s.lastCommittedBlockHeight,
			Timestamp:           input.ResultsBlockHeader.Timestamp(),
			TransactionReceipts: myReceipts,
		})
	}

	s.logger.Info("committed transaction receipts for block height", log.BlockHeight(s.lastCommittedBlockHeight))

	return &services.CommitTransactionReceiptsOutput{
		NextDesiredBlockHeight:   s.lastCommittedBlockHeight + 1,
		LastCommittedBlockHeight: s.lastCommittedBlockHeight,
	}, nil
}

func (s *service) originatedFromMyPublicApi(removedTx *pendingTransaction) bool {
	return removedTx != nil && removedTx.gatewayPublicKey.Equal(s.config.NodePublicKey())
}
