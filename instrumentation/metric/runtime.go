package metric

import (
	"context"
	"runtime"
	"time"
)

type runtimeMetrics struct {
	heapAlloc *Gauge
	heapSys *Gauge
	gcCpuPercentage *Gauge
}

type runtimeReporter struct {
	metrics runtimeMetrics
}

func NewRuntimeReporter(ctx context.Context, metricFactory Factory) interface{} {
	r := &runtimeReporter{
		metrics: runtimeMetrics {
			heapAlloc: metricFactory.NewGauge("Runtime.HeapAlloc"),
			heapSys: metricFactory.NewGauge("Runtime.HeapSys"),
			gcCpuPercentage: metricFactory.NewGauge("Runtime.GCCPUPercentage"),
		},
	}

	go r.startReporting(ctx)

	return r
}

func (r *runtimeReporter) startReporting(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			r.reportRuntimeMetrics()
		}
	}
}

func (r *runtimeReporter) reportRuntimeMetrics() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	//TODO debug.ReadGCStats()?

	r.metrics.heapSys.Update(int64(mem.HeapSys))
	r.metrics.heapAlloc.Update(int64(mem.HeapAlloc))
	r.metrics.gcCpuPercentage.Update(int64(mem.GCCPUFraction * 100))
}
