package ethereum

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"math/big"
	"strings"
)

var LogTag = log.Service("crosschain-connector")

type service struct {
	connection       adapter.EthereumConnection
	logger           log.BasicLogger
	timestampFetcher TimestampFetcher
	config           config.EthereumCrosschainConnectorConfig
}

func NewEthereumCrosschainConnector(connection adapter.EthereumConnection, config config.EthereumCrosschainConnectorConfig, parent log.BasicLogger) services.CrosschainConnector {
	logger := parent.WithTags(LogTag)
	s := &service{
		connection:       connection,
		timestampFetcher: NewTimestampFetcher(NewBlockTimestampFetcher(connection), logger),
		logger:           logger,
		config:           config,
	}
	return s
}

func NewEthereumCrosschainConnectorWithFakeTSF(connection adapter.EthereumConnection, config config.EthereumCrosschainConnectorConfig, parent log.BasicLogger) services.CrosschainConnector {
	logger := parent.WithTags(LogTag)
	s := &service{
		connection:       connection,
		timestampFetcher: NewTimestampFetcher(NewFakeBlockAndTimestampGetter(logger), logger),
		logger:           logger,
		config:           config,
	}
	return s
}

func (s *service) EthereumCallContract(ctx context.Context, input *services.EthereumCallContractInput) (*services.EthereumCallContractOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	var ethereumBlockNumber *big.Int
	var err error

	if input.EthereumBlockNumber == 0 { // caller specified the latest block number possible
		ethereumBlockNumber, err = getFinalitySafeBlockNumber(ctx, input.ReferenceTimestamp, s.timestampFetcher, s.config)
	} else { // caller specified a non-zero block number
		ethereumBlockNumber = new(big.Int).SetUint64(input.EthereumBlockNumber)
		err = verifyBlockNumberIsFinalitySafe(ctx, input.EthereumBlockNumber, input.ReferenceTimestamp, s.timestampFetcher, s.config)
	}
	if err != nil {
		return nil, err
	}

	if ethereumBlockNumber != nil { // simulator returns nil from GetBlockByTimestamp
		logger.Info("calling contract from ethereum",
			log.String("address", input.EthereumContractAddress),
			log.Uint64("requested-block", input.EthereumBlockNumber),
			log.Uint64("actual-block-requested", ethereumBlockNumber.Uint64()))

	}

	address, err := hexutil.Decode(input.EthereumContractAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode the contract address %s", input.EthereumContractAddress)
	}

	output, err := s.connection.CallContract(ctx, address, input.EthereumAbiPackedInputArguments, ethereumBlockNumber)
	if err != nil {
		return nil, errors.Wrap(err, "ethereum call failed")
	}

	return &services.EthereumCallContractOutput{
		EthereumAbiPackedOutput: output,
	}, nil
}

func (s *service) EthereumGetTransactionLogs(ctx context.Context, input *services.EthereumGetTransactionLogsInput) (*services.EthereumGetTransactionLogsOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("getting transaction logs", log.String("contract-address", input.EthereumContractAddress), log.String("event", input.EthereumEventName), log.Transaction(primitives.Sha256(input.EthereumTxhash)))

	ethereumTxHash, err := hexutil.Decode(input.EthereumTxhash)
	if err != nil {
		return nil, err
	}

	parsedABI, err := abi.JSON(strings.NewReader(input.EthereumJsonAbi))
	if err != nil {
		return nil, err
	}

	eventABI, found := parsedABI.Events[input.EthereumEventName]
	if !found {
		return nil, errors.Errorf("event with name '%s' not found in given ABI", input.EthereumEventName)
	}

	logs, err := s.connection.GetTransactionLogs(ctx, ethereumTxHash, eventABI.Id().Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting logs for Ethereum txhash %s of contract %s", input.EthereumTxhash, input.EthereumContractAddress)
	}

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/597): support multiple logs
	if len(logs) != 1 {
		return nil, errors.Errorf("expected exactly one log entry for txhash %s of contract %s but got %d", input.EthereumTxhash, input.EthereumContractAddress, len(logs))
	}

	ethereumBlockNumber := logs[0].BlockNumber
	ethereumTxIndex := logs[0].TxIndex
	err = verifyBlockNumberIsFinalitySafe(ctx, ethereumBlockNumber, input.ReferenceTimestamp, s.timestampFetcher, s.config)
	if err != nil {
		return nil, err
	}

	output, err := repackEventABIWithTopics(eventABI, logs[0])
	if err != nil {
		return nil, err
	}

	return &services.EthereumGetTransactionLogsOutput{
		EthereumAbiPackedOutputs: [][]byte{output},
		EthereumBlockNumber:      ethereumBlockNumber,
		EthereumTxindex:          ethereumTxIndex,
	}, nil
}

func (s *service) EthereumGetBlockNumber(ctx context.Context, input *services.EthereumGetBlockNumberInput) (*services.EthereumGetBlockNumberOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("getting current safe Ethereum block number")

	ethereumBlockNumber, err := getFinalitySafeBlockNumber(ctx, input.ReferenceTimestamp, s.timestampFetcher, s.config)
	if err != nil {
		return nil, err
	}

	if ethereumBlockNumber == nil {
		return nil, errors.Errorf("failed getting an actual current block number from Ethereum") // note: the geth simulator does not support this API
	}

	return &services.EthereumGetBlockNumberOutput{
		EthereumBlockNumber: ethereumBlockNumber.Uint64(),
	}, nil
}
