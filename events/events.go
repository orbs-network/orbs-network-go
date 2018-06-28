package events

import (
	"fmt"
	"time"
	"sync"
)

type Events interface {
	FinishedConsensusRound()
	ConsensusError(err error)
}

type ObservableEvents interface {
	Events

	WaitForConsensusRounds(roundsToWait int)
}

type Latch interface {
	Events

	WaitForConsensusRound()
}

type latch struct {
	cond *sync.Cond
}

func NewLatch() Latch {
	return &latch{}
}

func (l *latch) WaitForConsensusRound() {
	mutex := &sync.Mutex{}
	mutex.Lock()
	l.cond = sync.NewCond(mutex)
	l.cond.Wait()
}


func (l *latch) FinishedConsensusRound() {
	if l.cond != nil {
		l.cond.Broadcast()
		l.cond = nil
	}
}

func (l *latch) ConsensusError(err error) () {
}

type BufferingEvents interface {
	Events

	Flush()
}

type bufferingEvents struct {
	loggedEvents []string
	name         string
}

func NewBufferingEvents(name string) BufferingEvents {
	e := bufferingEvents{name: name}
	e.log("Start of log")
	return &e
}

func (e *bufferingEvents) Flush() {
	for _, line := range e.loggedEvents {
		println(line)
	}
}

func (e *bufferingEvents) FinishedConsensusRound() {
	e.log("Finished consensus round")
}

func (e *bufferingEvents) ConsensusError(err error) () {
	e.log(fmt.Sprintf("Error during consensus: %s", err))
}

func (e *bufferingEvents) log(message string) {
	e.loggedEvents = append(e.loggedEvents, fmt.Sprintf("[%s] [%s]: %s", e.name, time.Now().Format("15:04:05.99999999"), message))
}

type compositeEvents struct {
	children []Events
}

func NewCompositeEvents(children []Events) Events {
	return &compositeEvents{children: children}
}

func (e *compositeEvents) FinishedConsensusRound() {
	for _, child := range e.children {
		child.FinishedConsensusRound()
	}
}

func (e *compositeEvents) ConsensusError(err error) () {
	for _, child := range e.children {
		child.ConsensusError(err)
	}
}
