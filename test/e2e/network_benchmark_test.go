package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"testing"
)

func TestE2EStress(t *testing.T) {
	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 50)

	h := newHarness()
	defer h.gracefulShutdown()

	NUMBER_OF_TRANSACTIONS := 10000
	TRANSACTIONS_PER_SECOND := float64(1000)

	for i := 0; i < NUMBER_OF_TRANSACTIONS; i++ {
		if err := limiter.Wait(context.TODO()); err == nil {
			wg.Add(1)

			go func() {
				defer wg.Done()

				signerKeyPair := keys.Ed25519KeyPairForTests(5)
				targetAddress := builders.AddressForEd25519SignerForTests(6)
				amount := uint64(rand.Intn(10))

				transfer := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(amount, targetAddress).Builder()
				_, err := h.sendTransaction(transfer)
				if err != nil {
					t.Fatalf("error sending transaction %s\n", err)
				}
			}()
		} else {
			t.Fatalf("error %s\n", err)
		}
	}

	wg.Wait()

	res, _ := http.Get(h.absoluteUrlFor("/metrics"))
	bytes, _ := ioutil.ReadAll(res.Body)
	fmt.Println(string(bytes))

	metrics := make(map[string]map[string]interface{})
	err := json.Unmarshal(bytes, &metrics)

	require.NoError(t, err)

	txCount := metrics["TransactionPool.CommittedPool.TransactionCount"]["Value"]
	require.EqualValues(t, NUMBER_OF_TRANSACTIONS, txCount)

	ratePerSecond := metrics["TransactionPool.RatePerSecond"]["Rate"].(float64)

	require.Condition(t, func() (success bool) {
		return ratePerSecond >= TRANSACTIONS_PER_SECOND
	})
}
