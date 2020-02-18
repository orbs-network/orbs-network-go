package sdk

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSDKEnvGetBlockCommittee_ReturnSameCommitteeAsConfig(t *testing.T) {

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {

			harness := newVmHarness(parent.Logger)

			receipts, err := harness.processSuccessfully(ctx,
				builders.Transaction().WithMethod("_Committee", "getOrderedCommittee").WithArgs().Build(),
			)
			require.NoError(t, err)
			argsArray, err := protocol.PackedOutputArgumentsToNatives(receipts[0].RawOutputArgumentArrayWithHeader())
			require.NoError(t, err)
			require.Len(t, argsArray[0], 5)
			var nodeAddresses []primitives.NodeAddress
			for _, nodeArg := range argsArray[0].([][]byte) {
				nodeAddresses = append(nodeAddresses, nodeArg)
			}
			require.ElementsMatch(t, testKeys.NodeAddressesForTests()[:5], nodeAddresses)
		})
	})

}

