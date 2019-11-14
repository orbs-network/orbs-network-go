package sdk

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/sdk/contracts/slices"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestVm_WorkingContractWithBools(t *testing.T) {
	sliceOfBools := []bool{true, false, true, false, false, true}
	sliceOfUint32s := []uint32{1, 10, 100, 1000, 10000, 100000, 3}
	sliceOfUint64s := []uint64{1, 10, 100, 1000, 10000, 100000, 3}
	sliceOfUint256s := []*big.Int{big.NewInt(1), big.NewInt(1000000), big.NewInt(555555555555)}
	sliceOfStrings := []string{"picture", "yourself", "in", "a", "boat", "on", "a", "river"}
	sliceOfByteses := [][]byte{{0x11, 0x12}, {0xa, 0xb, 0xc, 0xd}, {0x1, 0x2}}
	sliceOfBytes20s := [][20]byte{{0xaa, 0xbb}, {0x11, 0x12}, {0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01}, {0x1, 0x2}}
	sliceOfBytes32s := [][32]byte{{0x11, 0x12}, {0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04,
		0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x01, 0x02, 0x03, 0x04}, {0xa, 0xb, 0xc, 0xd}, {0x1, 0x2}}


	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)
			harness.repository.Register(ContractName, slices.PUBLIC, slices.SYSTEM, nil)

			receipts, err := harness.processSuccessfully(ctx,
				generateDeployTx(),
				builders.Transaction().WithMethod(ContractName, "get").WithArgs().Build(),
				builders.Transaction().WithMethod(ContractName, "check").
				WithArgs(sliceOfBools, sliceOfUint32s, sliceOfUint64s, sliceOfUint256s, sliceOfStrings, sliceOfByteses, sliceOfBytes20s, sliceOfBytes32s).Build(),
			)
			require.NoError(t, err)

			// check output tx1
			argsArray, err := protocol.PackedOutputArgumentsToNatives(receipts[1].RawOutputArgumentArrayWithHeader())
			require.NoError(t, err)
			require.EqualValues(t, sliceOfBools, argsArray[0])
			require.EqualValues(t, sliceOfUint32s, argsArray[1])
			require.EqualValues(t, sliceOfUint64s, argsArray[2])
			require.EqualValues(t, sliceOfUint256s, argsArray[3])
			require.EqualValues(t, sliceOfStrings, argsArray[4])
			require.EqualValues(t, sliceOfByteses, argsArray[5])
			require.EqualValues(t, sliceOfBytes20s, argsArray[6])
			require.EqualValues(t, sliceOfBytes32s, argsArray[7])
			// check output tx2
			argsArray, err = protocol.PackedOutputArgumentsToNatives(receipts[2].RawOutputArgumentArrayWithHeader())
			require.NoError(t, err)
			require.True(t, argsArray[0].(bool), "should return true showing equality: err message %s", argsArray[1].(string))
			require.Empty(t, argsArray[1].(string))
		})
	})
}
