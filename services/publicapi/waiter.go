// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package publicapi

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"sync"
)

type waiterChannel struct {
	c chan interface{}
	k string
}

type waiterChannels map[*waiterChannel]*waiterChannel

type waiter struct {
	mutex sync.Mutex
	m     map[string]waiterChannels
}

func newWaiter() *waiter {
	return &waiter{
		mutex: sync.Mutex{},
		m:     make(map[string]waiterChannels),
	}
}

func (w *waiter) add(k string) *waiterChannel {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	var wcs waiterChannels
	var exists bool
	if wcs, exists = w.m[k]; !exists {
		wcs = make(waiterChannels)
		w.m[k] = wcs
	}
	wc := &waiterChannel{make(chan interface{}, 1), k} // channel is buffered for quick release
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

func (w *waiter) wait(ctx context.Context, wc *waiterChannel) (interface{}, error) {
	select {
	case <-ctx.Done():
		w.deleteByChannel(wc)
		return nil, errors.Wrapf(ctx.Err(), "waiting aborted due to context termination for key %s", hex.EncodeToString([]byte(wc.k)))
	case response, open := <-wc.c: // intentional not close channel here
		if !open {
			return nil, errors.Errorf("waiting aborted")
		}
		return response, nil
	}
}

func (w *waiter) complete(k string, wo interface{}) {
	for wc := range w._deleteByKey(k) {
		wc.c <- wo
		close(wc.c)
	}
}
