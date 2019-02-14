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
	heapIdle        *Gauge
	heapReleased    *Gauge
	heapInuse       *Gauge
	heapObjects     *Gauge
	gcCpuPercentage *Gauge
	numGc           *Gauge
	numGoroutine    *Gauge
}

type runtimeReporter struct {
	metrics runtimeMetrics
}

func NewRuntimeReporter(ctx context.Context, metricFactory Factory, logger log.BasicLogger) interface{} {
	r := &runtimeReporter{
		metrics: runtimeMetrics{
			heapAlloc:       metricFactory.NewGauge("Runtime.HeapAlloc"),
			heapSys:         metricFactory.NewGauge("Runtime.HeapSys"),
			heapIdle:        metricFactory.NewGauge("Runtime.HeapIdle"),
			heapReleased:    metricFactory.NewGauge("Runtime.HeapReleased"),
			heapInuse:       metricFactory.NewGauge("Runtime.HeapInuse"),
			heapObjects:     metricFactory.NewGauge("Runtime.HeapObjects"),
			gcCpuPercentage: metricFactory.NewGauge("Runtime.GCCPUPercentage"),
			numGc:           metricFactory.NewGauge("Runtime.NumGc"),
			numGoroutine:    metricFactory.NewGauge("Runtime.NumGoroutine"),
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

	r.metrics.heapSys.Update(int64(mem.HeapSys))
	r.metrics.heapAlloc.Update(int64(mem.HeapAlloc))
	r.metrics.heapIdle.Update(int64(mem.HeapIdle))
	r.metrics.heapReleased.Update(int64(mem.HeapReleased))
	r.metrics.heapInuse.Update(int64(mem.HeapInuse))
	r.metrics.heapObjects.Update(int64(mem.HeapObjects))
	r.metrics.gcCpuPercentage.Update(int64(mem.GCCPUFraction * 100))
	r.metrics.numGc.Update(int64(mem.NumGC))
	r.metrics.numGoroutine.Update(int64(runtime.NumGoroutine()))
}
