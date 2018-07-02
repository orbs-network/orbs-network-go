package loopcontrol

import (
	"sync"
	"github.com/orbs-network/orbs-network-go/events"
)

type LoopControl interface {
	NewLoop(name string, tickFunc func())
}

type BreakingLoopContext interface {
	Brake()
	Tick()
	Release()
}

type BrakingLoop interface {
	LoopControl
	LatchFor(name string) BreakingLoopContext
}

type brakingLoop struct {
	context *breakingLoopContext
	events  events.Events

	loops sync.Map
}

type breakingLoopContext struct {
	tickCond     *sync.Cond
	loopCond     *sync.Cond
	brakeCond    *sync.Cond
	brakeEnabled bool
	events       events.Events
}

func NewBrakingLoop(events events.Events) BrakingLoop {
	return &brakingLoop{events: events}
}

func (l *brakingLoop) NewLoop(name string, tickFunc func()) {
	c := l.createOrGetContext(name)

	for {

		if c.brakeEnabled {
			c.brakeCond.Signal()
			c.loopCond.Wait()
		}

		c.events.Report("enter_" + name)

		tickFunc()

		c.tickCond.Signal()

		c.events.Report("exit_" + name)
	}
}

func (l *brakingLoop) LatchFor(name string) BreakingLoopContext {
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
