// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	uptime          *Gauge
}

type runtimeReporter struct {
	metrics runtimeMetrics
	started time.Time
}

func NewRuntimeReporter(ctx context.Context, metricFactory Factory, logger log.BasicLogger) interface{} {
	r := &runtimeReporter{
		metrics: runtimeMetrics{
			heapAlloc:       metricFactory.NewGauge("Runtime.HeapAlloc.Bytes"),
			heapSys:         metricFactory.NewGauge("Runtime.HeapSys.Bytes"),
			heapIdle:        metricFactory.NewGauge("Runtime.HeapIdle.Bytes"),
			heapReleased:    metricFactory.NewGauge("Runtime.HeapReleased.Bytes"),
			heapInuse:       metricFactory.NewGauge("Runtime.HeapInuse.Bytes"),
			heapObjects:     metricFactory.NewGauge("Runtime.HeapObjects.Number"),
			gcCpuPercentage: metricFactory.NewGauge("Runtime.GCCPUPercentage.Number"),
			numGc:           metricFactory.NewGauge("Runtime.NumGc.Number"),
			numGoroutine:    metricFactory.NewGauge("Runtime.NumGoroutine.Number"),
			uptime:          metricFactory.NewGauge("Runtime.Uptime.Seconds"),
		},
		started: time.Now(),
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
	r.metrics.uptime.Update(int64(time.Since(r.started).Seconds()))
}
