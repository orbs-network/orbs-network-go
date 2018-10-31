package metric

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"runtime"
	"time"
)

type runtimeMetrics struct {
	heapAlloc       *Gauge
	heapSys         *Gauge
	gcCpuPercentage *Gauge
}

type runtimeReporter struct {
	metrics runtimeMetrics
}

func NewRuntimeReporter(ctx context.Context, metricFactory Factory, logger log.BasicLogger) interface{} {
	r := &runtimeReporter{
		metrics: runtimeMetrics{
			heapAlloc:       metricFactory.NewGauge("Runtime.HeapAlloc"),
			heapSys:         metricFactory.NewGauge("Runtime.HeapSys"),
			gcCpuPercentage: metricFactory.NewGauge("Runtime.GCCPUPercentage"),
		},
	}

	r.startReporting(ctx, logger)

	return r
}

func (r *runtimeReporter) startReporting(ctx context.Context, logger log.BasicLogger) {
	synchronization.NewPeriodicalTrigger(ctx, 5*time.Second, logger, func() {
		r.reportRuntimeMetrics()
	}, nil)
}

func (r *runtimeReporter) reportRuntimeMetrics() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	//TODO debug.ReadGCStats()?

	r.metrics.heapSys.Update(int64(mem.HeapSys))
	r.metrics.heapAlloc.Update(int64(mem.HeapAlloc))
	r.metrics.gcCpuPercentage.Update(int64(mem.GCCPUFraction * 100))
}
