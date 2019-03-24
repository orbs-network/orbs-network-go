// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"math/rand"
	"os"
	"sync"
)

var mutex = sync.Mutex{}
var lastPort = 0

// Random ports encourage flakiness. Instead, when possible, pass 0 as the TCP port and then retrieve the port for client usage:
//  listener, err := net.Listen("tcp", "127.0.0.1:0")
//  port := listener.Addr().(*net.TCPAddr).Port
func RandomPort() int {
	mutex.Lock()
	defer mutex.Unlock()

	if lastPort == 0 {
		src := rand.NewSource(int64(os.Getpid()))
		lastPort = rand.New(src).Intn(30000) + 25111
	} else {
		lastPort++
	}

	return lastPort
}
