package e2e

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"golang.org/x/time/rate"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"testing"
)

func BenchmarkE2ENetwork(b *testing.B) {

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(1000, 100)

	h := newHarness()
	defer h.gracefulShutdown()

	for i := 0; i < b.N; i++ {
		if err := limiter.Wait(context.TODO()); err == nil {
			wg.Add(1)

			go func() {
				defer wg.Done()

				signerKeyPair := keys.Ed25519KeyPairForTests(5)
				targetAddress := builders.AddressForEd25519SignerForTests(6)
				transfer := builders.TransferTransaction().WithEd25519Signer(signerKeyPair).WithAmountAndTargetAddress(uint64(rand.Intn(i)), targetAddress).Builder()
				_, err := h.sendTransaction(transfer)
				if err != nil {
					fmt.Printf("error sending transaction %s\n", err)
				}
			}()

		}

	}

	res, _ := http.Get(h.absoluteUrlFor("/metrics"))
	bytes, _ := ioutil.ReadAll(res.Body)
	println(string(bytes))

	wg.Wait()

}