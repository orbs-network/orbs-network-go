package test

import (
	"math/rand"
	"sync"
	"time"
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
		src := rand.NewSource(time.Now().UnixNano())
		lastPort = rand.New(src).Intn(30000) + 25111
	} else {
		lastPort++
	}

	return lastPort
}
