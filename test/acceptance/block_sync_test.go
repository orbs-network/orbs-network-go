package acceptance

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"math/rand"
	"strconv"
	"testing"
)

func TestBlockSync(t *testing.T) {
	testId := "acceptance-BlockSync-" + strconv.FormatUint(rand.Uint64(), 10)
	defer harness.ReportTestId(t, testId)

	harness.WithNetwork(t, testId, 2, harness.WithAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS), func(network harness.AcceptanceTestNetwork) {
		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 0 failed: %s", err)
		}

		network.GossipTransport().Fail(func(data *adapter.TransportData) bool {
			header := gossipmessages.HeaderReader(data.Payloads[0])
			return header.IsTopicBenchmarkConsensus() && header.BenchmarkConsensus() == consensus.BENCHMARK_CONSENSUS_COMMIT && data.SenderPublicKey.Equal(keys.Ed25519KeyPairForTests(0).PublicKey())
		})

		//require.EqualValues(t, 17, <-network.CallGetBalance(0), "eventual getBalance result on leader")

		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(5); err != nil {
			t.Errorf("waiting for block on node 1 failed: %s", err)
		}
		//require.EqualValues(t, 17, <-network.CallGetBalance(1), "eventual getBalance result on non leader")

	})
}
