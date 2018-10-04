package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

var LogTag = log.Service("public-api")

type Config interface {
	SendTransactionTimeout() time.Duration
	VirtualChainId() primitives.VirtualChainId
}

type txResponse struct {
	transactionStatus  protocol.TransactionStatus
	transactionReceipt *protocol.TransactionReceipt
	blockHeight        primitives.BlockHeight
	blockTimestamp     primitives.TimestampNano
}

type service struct {
	ctx             context.Context
	config          Config
	transactionPool services.TransactionPool
	virtualMachine  services.VirtualMachine
	blockStorage    services.BlockStorage
	logger          log.BasicLogger

	waiter *waiter
}

func NewPublicApi(
	ctx context.Context,
	config Config,
	transactionPool services.TransactionPool,
	virtualMachine services.VirtualMachine,
	blockStorage services.BlockStorage,
	logger log.BasicLogger,
) services.PublicApi {
	s := &service{
		ctx:             ctx,
		config:          config,
		transactionPool: transactionPool,
		virtualMachine:  virtualMachine,
		blockStorage:    blockStorage,
		logger:          logger.WithTags(LogTag),

		waiter: newWaiter(ctx),
	}

	transactionPool.RegisterTransactionResultsHandler(s)

	return s
}

func (s *service) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	for _, txReceipt := range input.TransactionReceipts {
		s.logger.Info("transaction reported as committed", log.String("flow", "checkpoint"), log.Stringable("txHash", txReceipt.Txhash()))
		s.waiter.complete(txReceipt.Txhash().KeyForMap(),
			&waiterObject{&txResponse{
				transactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
				transactionReceipt: txReceipt,
				blockHeight:        input.BlockHeight,
				blockTimestamp:     input.Timestamp,
			}})
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}

func (s *service) HandleTransactionError(input *handlers.HandleTransactionErrorInput) (*handlers.HandleTransactionErrorOutput, error) {
	s.logger.Info("transaction reported as errored", log.String("flow", "checkpoint"), log.Stringable("txHash", input.Txhash), log.Stringable("tx-status", input.TransactionStatus))
	s.waiter.complete(input.Txhash.KeyForMap(),
		&waiterObject{&txResponse{
			transactionStatus:  input.TransactionStatus,
			transactionReceipt: nil,
			blockHeight:        input.BlockHeight,
			blockTimestamp:     input.BlockTimestamp,
		}})
	return &handlers.HandleTransactionErrorOutput{}, nil
}

func isTransactionRequestValid(config Config, tx *protocol.Transaction) protocol.TransactionStatus {
	if config.VirtualChainId() != tx.VirtualChainId() {
		return protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH
	}

	if primitives.ProtocolVersion(1) != tx.ProtocolVersion() {
		return protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION
	}

	return protocol.TRANSACTION_STATUS_RESERVED // used as an OK
}

func translateTxStatusToResponseCode(txStatus protocol.TransactionStatus) protocol.RequestStatus {
	switch txStatus {
	case protocol.TRANSACTION_STATUS_COMMITTED:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.TRANSACTION_STATUS_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_PENDING:
		return protocol.REQUEST_STATUS_IN_PROCESS
	case protocol.TRANSACTION_STATUS_NO_RECORD_FOUND:
		return protocol.REQUEST_STATUS_NOT_FOUND
	case protocol.TRANSACTION_STATUS_REJECTED_UNSUPPORTED_VERSION:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_MISMATCH:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_SIGNATURE_MISMATCH:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_UNKNOWN_SIGNER_SCHEME:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_VIRTUAL_CHAIN_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.TRANSACTION_STATUS_REJECTED_CONGESTION:
		return protocol.REQUEST_STATUS_CONGESTION
	}
	return protocol.REQUEST_STATUS_RESERVED
}

func translateExecutionStatusToResponseCode(executionResult protocol.ExecutionResult) protocol.RequestStatus {
	switch executionResult {
	case protocol.EXECUTION_RESULT_SUCCESS:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT:
		return protocol.REQUEST_STATUS_COMPLETED
	case protocol.EXECUTION_RESULT_ERROR_INPUT:
		return protocol.REQUEST_STATUS_REJECTED
	case protocol.EXECUTION_RESULT_ERROR_UNEXPECTED:
		return protocol.REQUEST_STATUS_SYSTEM_ERROR
	}
	return protocol.REQUEST_STATUS_RESERVED

}
