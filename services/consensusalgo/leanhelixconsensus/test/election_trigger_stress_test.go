package test

import (
	"context"
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
	"time"
)

func buildElectionTrigger(ctx context.Context, logger log.BasicLogger, timeout time.Duration) interfaces.ElectionTrigger {
	et := leanhelixconsensus.NewExponentialBackoffElectionTrigger(logger, timeout, nil)
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("BAD")
				return
			case trigger := <-et.ElectionChannel():
				trigger(ctx)
			}
		}
	}()

	return et
}

func TestStress_FrequentRegisters(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		et := buildElectionTrigger(tctx, log.DefaultTestingLogger(t), 1*time.Microsecond)

		var counter int32
		for h := primitives.BlockHeight(1); h < primitives.BlockHeight(10000); h++ {
			et.RegisterOnElection(tctx, h, 0, nil)
			counter++
			time.Sleep(1 * time.Microsecond)
		}
		t.Log(counter)
	})

}
