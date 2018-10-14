package metric

import (
	"fmt"
	"github.com/VividCortex/ewma"
	"sync"
	"time"
)

var tickInterval = 1 * time.Second

type Rate struct {
	namedMetric
	movingAverage ewma.MovingAverage

	m          sync.Mutex
	runningSum int64
	nextTick   time.Time
}

func newRate(name string) *Rate {
	return &Rate{
		namedMetric:   namedMetric{name: name},
		movingAverage: ewma.NewMovingAverage(),
		nextTick:      time.Now().Add(tickInterval),
	}
}

func (r *Rate) Export() interface{} {
	return struct {
		Name     string
		Rate     float64
		Interval time.Duration
	}{
		r.name,
		r.movingAverage.Value(),
		tickInterval,
	}
}

func (r *Rate) String() string {
	return fmt.Sprintf("metric %s: %f per %s\n", r.name, r.movingAverage.Value(), tickInterval)
}

func (r *Rate) Measure(eventCount int64) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.nextTick.Before(time.Now()) {
		r.movingAverage.Add(float64(r.runningSum))
		r.runningSum = 0
		r.nextTick = r.nextTick.Add(tickInterval)
	}

	r.runningSum += eventCount
}

func (r *Rate) Reset() {
	r.m.Lock()
	defer r.m.Unlock()

	r.movingAverage = ewma.NewMovingAverage()
}
