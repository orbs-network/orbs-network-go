package publicapi

import (
	"context"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type waiterObject struct {
	payload interface{}
}

type waiterChannel struct {
	c chan *waiterObject
	k string
}

type waiterChannels map[*waiterChannel]*waiterChannel

type waiter struct {
	ctx   context.Context
	mutex sync.Mutex
	m     map[string]waiterChannels
}

func newWaiter(ctx context.Context) *waiter {
	// TODO supervise
	return &waiter{
		ctx:   ctx,
		mutex: sync.Mutex{},
		m:     make(map[string]waiterChannels),
	}
}

func (w *waiter) add(k string) *waiterChannel {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	var wcs waiterChannels
	exists := false
	if wcs, exists = w.m[k]; !exists {
		wcs = make(waiterChannels)
		w.m[k] = wcs
	}
	wc := &waiterChannel{make(chan *waiterObject, 1), k} // channel is buffered for quick release
	wcs[wc] = wc

	return wc
}

func (w *waiter) _deleteByKey(k string) waiterChannels { // this is internal function only
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if wcs, exists := w.m[k]; exists {
		delete(w.m, k)
		return wcs
	}
	return nil
}

func (w *waiter) deleteByChannel(wc *waiterChannel) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if wcs, exists := w.m[wc.k]; exists {
		if _, existsC := wcs[wc]; existsC {
			delete(wcs, wc)
			if len(wcs) == 0 { // if we were the last ones clean up
				delete(w.m, wc.k)
			}
			close(wc.c)
		}
	}
}

func (w *waiter) wait(wc *waiterChannel, duration time.Duration) (*waiterObject, error) {
	timer := synchronization.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-w.ctx.Done(): // currently ctx is global so only shutdown and no need to kill the map.
		//	w.deleteByChannel(wc)
		return nil, errors.Errorf("shutting down")
	case <-timer.C:
		w.deleteByChannel(wc)
		return nil, errors.Errorf("timed out waiting for result %s", hex.EncodeToString([]byte(wc.k)))
	case response, open := <-wc.c: // intentional not close channel here
		if !open {
			return nil, errors.Errorf("waiting aborted")
		}
		return response, nil
	}
}

func (w *waiter) complete(k string, wo *waiterObject) {
	for wc := range w._deleteByKey(k) {
		wc.c <- wo
		close(wc.c)
	}
}
