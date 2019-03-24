// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"fmt"
	"github.com/VividCortex/ewma"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync"
	"time"
)

var hardCodedTickInterval = 1 * time.Second // this cannot really be changed as the EWMA library doesn't work well with sub-second intervals

type Rate struct {
	namedMetric
	movingAverage ewma.MovingAverage

	m          sync.Mutex
	runningSum int64
	nextTick   time.Time
}

type rateExport struct {
	Name     string
	Rate     float64
	Interval float64
}

func newRate(name string) *Rate {
	return newRateWihStart(name, time.Now())
}

func newRateWihStart(name string, start time.Time) *Rate {
	return &Rate{
		namedMetric:   namedMetric{name: name},
		movingAverage: ewma.NewMovingAverage(),
		nextTick:      start.Add(hardCodedTickInterval),
	}
}

func (r *Rate) Export() exportedMetric {
	return r.export()
}

func (r *Rate) export() rateExport {
	r.m.Lock()
	defer r.m.Unlock()
	r.maybeRotate()

	return rateExport{
		r.name,
		r.movingAverage.Value(),
		toMillis(hardCodedTickInterval.Nanoseconds()),
	}
}

func (r *Rate) String() string {
	return fmt.Sprintf("metric %s: %f per %s\n", r.name, r.movingAverage.Value(), hardCodedTickInterval)
}

func (r *Rate) Measure(eventCount int64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.maybeRotate()
	r.runningSum += eventCount
}

func (r *Rate) maybeRotate() {
	r.maybeRotateAsOf(time.Now())
}

func (r *Rate) maybeRotateAsOf(asOf time.Time) {
	if r.nextTick.Before(asOf) {
		r.movingAverage.Add(float64(r.runningSum))
		r.runningSum = 0
		r.nextTick = r.nextTick.Add(hardCodedTickInterval)
	}
}

func (r *Rate) Reset() {
	r.m.Lock()
	defer r.m.Unlock()

	r.movingAverage = ewma.NewMovingAverage()
}

func (r rateExport) LogRow() []*log.Field {
	return []*log.Field{
		log.String("metric", r.Name),
		log.String("metric-type", "rate"),
		log.Float64("rate", r.Rate),
		log.Float64("interval", r.Interval),
	}
}
