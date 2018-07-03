package instrumentation

import (
	"sync"
	"github.com/orbs-network/orbs-network-go/instrumentation"
)

type BrakingLoopContext interface {
	Brake()
	Tick()
	Release()
}

type BrakingLoop interface {
	instrumentation.LoopControl
	LatchFor(name string) BrakingLoopContext
}

type brakingLoop struct {
	context *breakingLoopContext
	events  instrumentation.Reporting

	loops sync.Map
}

type breakingLoopContext struct {
	tickCond     *sync.Cond
	loopCond     *sync.Cond
	brakeCond    *sync.Cond
	brakeEnabled bool
	events       instrumentation.Reporting
}

func NewBrakingLoop(events instrumentation.Reporting) BrakingLoop {
	return &brakingLoop{events: events}
}

func (l *brakingLoop) NewLoop(name string, tickFunc func()) {
	c := l.createOrGetContext(name)

	for {

		if c.brakeEnabled {
			c.brakeCond.Signal()
			c.loopCond.Wait()
		}

		c.events.Info("enter_" + name)

		tickFunc()

		c.tickCond.Signal()

		c.events.Info("exit_" + name)
	}
}

func (l *brakingLoop) LatchFor(name string) BrakingLoopContext {
	return l.createOrGetContext(name)
}

func (c *breakingLoopContext) Brake() {
	c.loopCond.L.Lock()
	c.tickCond.L.Lock()
	c.brakeCond.L.Lock()
	c.brakeEnabled = true
	c.brakeCond.Wait()
}

func (c *breakingLoopContext) Tick() {
	if !c.brakeEnabled {
		panic("Tick without brake!")
	}
	c.loopCond.Signal()
	c.tickCond.Wait()
}

func (c *breakingLoopContext) Release() {
	c.brakeEnabled = false
	c.loopCond.Signal()
	c.tickCond.Signal()

}

func (l *brakingLoop) createOrGetContext(name string) *breakingLoopContext {
	context, _ := l.loops.LoadOrStore(name, &breakingLoopContext{
		tickCond: sync.NewCond(&sync.Mutex{}),
		loopCond: sync.NewCond(&sync.Mutex{}),
		brakeCond: sync.NewCond(&sync.Mutex{}),
		events: l.events,
	})

	c := context.(*breakingLoopContext) // casting because no generics :-(

	return c
}
