package sdk

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/sdk/contracts/fixed_bytes"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

const ContractName = "TestBytesContract"

func generateDeployTx() *protocol.SignedTransaction {
	return builders.Transaction().WithMethod("_Deployments", "deployService").
		WithArgs(ContractName, uint32(protocol.PROCESSOR_TYPE_NATIVE), []byte("irrelevant data - contract is already registered")).
		Build()
}

func TestVm_WorkingContractWithBytes20(t *testing.T) {
	bytes20 := [20]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, fixed_bytes.PUBLIC, fixed_bytes.SYSTEM, nil)

			receipts, err := harness.processSuccessfully(ctx,
				generateDeployTx(),
				builders.Transaction().WithMethod(ContractName, "setAddress").WithArgs(bytes20).Build(),
				builders.Transaction().WithMethod(ContractName, "getAddress").WithArgs().Build(),
			)

			require.NoError(t, err)
			argsArray := builders.PackedArgumentArrayDecode(receipts[2].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, bytes20, argsArray.ArgumentsIterator().NextArguments().Bytes20Value())
		})
	})
}

func TestVm_WorkingContractWithBytes32(t *testing.T) {
	bytes32 := [32]byte{0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, fixed_bytes.PUBLIC, fixed_bytes.SYSTEM, nil)

			receipts, err := harness.processSuccessfully(ctx,
				generateDeployTx(),
				builders.Transaction().WithMethod(ContractName, "setHash").WithArgs(bytes32).Build(),
				builders.Transaction().WithMethod(ContractName, "getHash").WithArgs().Build(),
			)

			require.NoError(t, err)
			argsArray := builders.PackedArgumentArrayDecode(receipts[2].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, bytes32, argsArray.ArgumentsIterator().NextArguments().Bytes32Value())
		})
	})
}

func TestVm_WorkingContractWithBigInt(t *testing.T) {
	tokenValue := big.NewInt(5000001000)

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, fixed_bytes.PUBLIC, fixed_bytes.SYSTEM, nil)

			receipts, err := harness.processSuccessfully(ctx,
				generateDeployTx(),
				builders.Transaction().WithMethod(ContractName, "setToken").WithArgs(tokenValue).Build(),
				builders.Transaction().WithMethod(ContractName, "getToken").WithArgs().Build(),
			)

			require.NoError(t, err)
			argsArray := builders.PackedArgumentArrayDecode(receipts[2].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, tokenValue, argsArray.ArgumentsIterator().NextArguments().Uint256Value())
		})
	})
}

func TestVm_WorkingContractWithBool(t *testing.T) {
	configEnabled := true

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, fixed_bytes.PUBLIC, fixed_bytes.SYSTEM, nil)

			receipts, err := harness.processSuccessfully(ctx,
				generateDeployTx(),
				builders.Transaction().WithMethod(ContractName, "setBool").WithArgs(configEnabled).Build(),
				builders.Transaction().WithMethod(ContractName, "getBool").WithArgs().Build(),
			)

			require.NoError(t, err)
			argsArray := builders.PackedArgumentArrayDecode(receipts[2].RawOutputArgumentArrayWithHeader())
			require.EqualValues(t, configEnabled, argsArray.ArgumentsIterator().NextArguments().BoolValue())
		})
	})
}
