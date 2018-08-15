package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func (s *service) GetTransactionsForOrdering(input *services.GetTransactionsForOrderingInput) (*services.GetTransactionsForOrderingOutput, error) {

	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
		return nil, err
	}

	out := &services.GetTransactionsForOrderingOutput{}
	transactions := s.pendingPool.getBatch(input.MaxNumberOfTransactions, input.MaxTransactionsSetSizeKb*1024)
	vctx := s.createValidationContext()

	transactionsForPreOrder := make([]*protocol.SignedTransaction, 0, input.MaxNumberOfTransactions)
	for _, tx := range transactions {
		if err := vctx.validateTransaction(tx); err != nil {
			s.log.Info("dropping invalid transaction", instrumentation.Error(err), instrumentation.Stringable("transaction", tx))
		} else {
			transactionsForPreOrder = append(transactionsForPreOrder, tx)
		}

		//else if alreadyCommitted := s.committedPool.get(tx); alreadyCommitted != nil {
		//	s.log.Info("dropping committed transaction", instrumentation.Stringable("transaction", tx))
		//}

	}

	//TODO handle error from vm
	preOrderResults, _ := s.virtualMachine.TransactionSetPreOrder(&services.TransactionSetPreOrderInput{
		SignedTransactions: transactionsForPreOrder,
	})

	for i := range transactionsForPreOrder {
		if preOrderResults.PreOrderResults[i] == protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			out.SignedTransactions = append(out.SignedTransactions, transactionsForPreOrder[i])
		}
	}

	// START OF THROWAWAY CODE TODO remove the following as soon as block storage can call CommitTransactionReceipts
	for _, tx := range out.SignedTransactions {
		s.pendingPool.remove(digest.CalcTxHash(tx.Transaction()))
	}
	// END OF THROWAWAY CODE

	return out, nil
}
