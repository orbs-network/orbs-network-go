// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestContextId_Simple(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		var CONTEXT_ID = []byte{0x17, 0x18}

		call := processCallInput().WithContextId(CONTEXT_ID).WithMethod("BenchmarkContract", "set").WithArgs(uint64(66)).Build()
		h.expectSdkCallMadeWithExecutionContextId(CONTEXT_ID)

		_, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		h.verifySdkCallMade(t)
	})
}

func TestContextId_MultipleGoroutines(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		var wg sync.WaitGroup
		h := newHarness(t)

		for i := 0; i < 20; i++ {
			wg.Add(1)
			var CONTEXT_ID = sdkContext.ContextId([]byte{0x17, byte(i + 17)})

			go func() {
				call := processCallInput().WithContextId(CONTEXT_ID).WithMethod("BenchmarkContract", "set").WithArgs(uint64(66)).Build()
				h.expectSdkCallMadeWithExecutionContextId(CONTEXT_ID)

				time.Sleep(5 * time.Millisecond)

				_, err := h.service.ProcessCall(ctx, call)
				require.NoError(t, err, "call should succeed")

				wg.Done()
			}()

		}

		wg.Wait()
		h.verifySdkCallMade(t)
	})
}
