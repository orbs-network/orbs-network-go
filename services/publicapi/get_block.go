package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) GetBlock(parentCtx context.Context, input *services.GetBlockInput) (*services.GetBlockOutput, error) {
	ctx := trace.NewContext(parentCtx, "PublicApi.GetBlock")

	if input.ClientRequest == nil {
		err := errors.Errorf("client request is nil")
		s.logger.Info("get block received missing input", log.Error(err))
		return nil, err
	}

	logger := s.logger.WithTags(trace.LogFieldFrom(ctx), log.BlockHeight(input.ClientRequest.BlockHeight()), log.String("flow", "checkpoint"))

	if _, err := validateRequest(s.config, input.ClientRequest.ProtocolVersion(), input.ClientRequest.VirtualChainId()); err != nil {
		logger.Info("get block received input failed", log.Error(err))
		return toGetBlockErrOutput(protocol.REQUEST_STATUS_BAD_REQUEST, 0, 0), err
	}

	logger.Info("get block request received")

	bpc, err := s.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{
		BlockHeight: input.ClientRequest.BlockHeight(),
	})
	if err != nil {
		logger.Info("block storage failed", log.Error(err))
		return toGetBlockErrOutput(protocol.REQUEST_STATUS_SYSTEM_ERROR, 0, 0), err
	}
	if bpc.BlockPair == nil {
		logger.Info("get block failed to get requested block height", log.BlockHeight(input.ClientRequest.BlockHeight()))
		bk, err2 := s.blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
		if err2 != nil {
			logger.Info("block storage failed while getting last block", log.Error(err2))
			return toGetBlockErrOutput(protocol.REQUEST_STATUS_SYSTEM_ERROR, 0, 0), err2
		}
		return toGetBlockErrOutput(protocol.REQUEST_STATUS_BAD_REQUEST, bk.LastCommittedBlockHeight, bk.LastCommittedBlockTimestamp), nil
	}

	return toGetBlockOutput(bpc.BlockPair), nil
}

func toGetBlockOutput(bpc *protocol.BlockPairContainer) *services.GetBlockOutput {
	signedTransactionBuilders := make([]*protocol.SignedTransactionBuilder, len(bpc.TransactionsBlock.SignedTransactions))
	for i, stx := range bpc.TransactionsBlock.SignedTransactions {
		signedTransactionBuilders[i] = protocol.SignedTransactionBuilderFromRaw(stx.Raw())
	}
	transactionReceiptBuilders := make([]*protocol.TransactionReceiptBuilder, len(bpc.ResultsBlock.TransactionReceipts))
	for i, txr := range bpc.ResultsBlock.TransactionReceipts {
		transactionReceiptBuilders[i] = protocol.TransactionReceiptBuilderFromRaw(txr.Raw())
	}
	contractStateDiffBuilders := make([]*protocol.ContractStateDiffBuilder, len(bpc.ResultsBlock.ContractStateDiffs))
	for i, csd := range bpc.ResultsBlock.ContractStateDiffs {
		contractStateDiffBuilders[i] = protocol.ContractStateDiffBuilderFromRaw(csd.Raw())
	}

	response := client.GetBlockResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    bpc.TransactionsBlock.Header.BlockHeight(),
			BlockTimestamp: bpc.TransactionsBlock.Header.Timestamp(),
		},
		TransactionsBlockHeader:   protocol.TransactionsBlockHeaderBuilderFromRaw(bpc.TransactionsBlock.Header.Raw()),
		TransactionsBlockMetadata: protocol.TransactionsBlockMetadataBuilderFromRaw(bpc.TransactionsBlock.Metadata.Raw()),
		SignedTransactions:        signedTransactionBuilders,
		TransactionsBlockProof:    protocol.TransactionsBlockProofBuilderFromRaw(bpc.TransactionsBlock.BlockProof.Raw()),
		ResultsBlockHeader:        protocol.ResultsBlockHeaderBuilderFromRaw(bpc.ResultsBlock.Header.Raw()),
		TransactionReceipts:       transactionReceiptBuilders,
		ContractStateDiffs:        contractStateDiffBuilders,
		ResultsBlockProof:         protocol.ResultsBlockProofBuilderFromRaw(bpc.ResultsBlock.BlockProof.Raw()),
	}

	return &services.GetBlockOutput{ClientResponse: response.Build()}
}

func toGetBlockErrOutput(status protocol.RequestStatus, height primitives.BlockHeight, nano primitives.TimestampNano) *services.GetBlockOutput {
	response := client.GetBlockResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  status,
			BlockHeight:    height,
			BlockTimestamp: nano,
		},
	}

	return &services.GetBlockOutput{ClientResponse: response.Build()}
}
