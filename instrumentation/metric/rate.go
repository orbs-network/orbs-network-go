// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"github.com/VividCortex/ewma"
	"sync"
	"time"
)

var hardCodedTickInterval = 1 * time.Second // this cannot really be changed as the EWMA library doesn't work well with sub-second intervals

type Rate struct {
	name 		  string
	movingAverage ewma.MovingAverage

	m          sync.Mutex
	runningSum int64
	nextTick   time.Time
}

type rateExport struct {
	Name          string
	RatePerSecond float64
}

func newRate(name string) *Rate {
	return newRateWihStart(name, time.Now())
}

func newRateWihStart(name string, start time.Time) *Rate {
	return &Rate{
		name:          name,
		movingAverage: ewma.NewMovingAverage(),
		nextTick:      start.Add(hardCodedTickInterval),
	}
}

func (r *Rate) Name() string {
	return r.name
}

func (r *Rate) Value() interface{} {
	return r.movingAverage.Value()
}

func (r *Rate) Rate() float64 {
	return r.movingAverage.Value()
}

func (r *Rate) Export() exportedMetric {
	r.m.Lock()
	defer r.m.Unlock()
	r.maybeRotate()

	return rateExport{
		r.name,
		r.Rate(),
	}
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
