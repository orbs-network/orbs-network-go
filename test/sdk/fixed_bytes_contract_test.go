package sdk

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/sdk/contracts/fixed_bytes"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

const ContractName = "TestBytesContract"

func TestVm_CanCompileContractWithFixedBytes(t *testing.T) {
	bytes20 := [20]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}
	bytes32 := [32]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}
	almostEmptyBytes20 := [20]byte{0x01}

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, fixed_bytes.PUBLIC, fixed_bytes.SYSTEM, nil)

			txs := []*protocol.SignedTransaction{
				//					builders.Transaction().WithMethod(WorkingContractName, WorkingMethodName).WithArgs(hash.Make32BytesWithFirstByte(6)).Build(),
				builders.Transaction().WithMethod("_Deployments", "deployService").
					WithArgs(ContractName, uint32(protocol.PROCESSOR_TYPE_NATIVE), []byte("irrelevant data - contract is already registered")).
					Build(),
				builders.Transaction().WithMethod(ContractName, "setAddress").WithArgs(bytes20).Build(),
				builders.Transaction().WithMethod(ContractName, "setHash").WithArgs(bytes32).Build(),
				builders.Transaction().WithMethod(ContractName, "getAddress").WithArgs().Build(),
				builders.Transaction().WithMethod(ContractName, "getHash").WithArgs().Build(),
				builders.Transaction().WithMethod(ContractName, "setAddress").WithArgs(almostEmptyBytes20).Build(),
				builders.Transaction().WithMethod(ContractName, "getAddress").WithArgs().Build(),
			}

			out, err := harness.vm.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
				CurrentBlockHeight:    1,
				CurrentBlockTimestamp: 66,
				SignedTransactions:    txs,
				BlockProposerAddress:  hash.Make32BytesWithFirstByte(5),
			})

			require.NoError(t, err)
			t.Log(out.StringContractStateDiffs())
			for i := 0; i < len(txs); i++ {
				executionResult := out.TransactionReceipts[i].ExecutionResult()
				require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS, executionResult, "tx %d should succeed. execution res was %s", i, executionResult)
			}

			argsArray := builders.PackedArgumentArrayDecode(out.TransactionReceipts[3].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, bytes20, argsArray.ArgumentsIterator().NextArguments().Bytes20Value())

			argsArray = builders.PackedArgumentArrayDecode(out.TransactionReceipts[4].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, bytes32, argsArray.ArgumentsIterator().NextArguments().Bytes32Value())

			argsArray = builders.PackedArgumentArrayDecode(out.TransactionReceipts[6].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, almostEmptyBytes20, argsArray.ArgumentsIterator().NextArguments().Bytes20Value())
		})
	})
}
