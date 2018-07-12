package instrumentation

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"sync"
	"time"
)

type Latch interface {
	instrumentation.Reporting

	WaitFor(message string)
}

type latch struct {
	cond       *sync.Cond
	waitingFor string
}

func NewLatch() Latch {
	return &latch{}
}

func (l *latch) WaitFor(message string) {
	l.waitingFor = message
	mutex := &sync.Mutex{}
	mutex.Lock()
	l.cond = sync.NewCond(mutex)
	l.cond.Wait()
}

func (l *latch) Info(message string) {
	if l.waitingFor == message && l.cond != nil {
		l.cond.Broadcast()
		l.cond = nil
		l.waitingFor = ""
	}
}

func (l *latch) Error(err error) {
	l.Info(err.Error())
}

type BufferedLog interface {
	instrumentation.Reporting

	Flush()
}

type bufferedLog struct {
	loggedEvents []string
	name         string
}

func NewBufferedLog(name string) BufferedLog {
	e := bufferedLog{name: name}
	e.log("Start of log")
	return &e
}

func (e *bufferedLog) Flush() {
	for _, line := range e.loggedEvents {
		println(line)
	}
}

func (e *bufferedLog) Info(message string) {
	e.log(message)
}

func (e *bufferedLog) Error(err error) {
	e.log(err.Error())
}

func (e *bufferedLog) log(message string) {
	e.loggedEvents = append(e.loggedEvents, fmt.Sprintf("[%s] [%s]: %s", e.name, time.Now().Format("15:04:05.99999999"), message))
}
