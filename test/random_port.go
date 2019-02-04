package test

import (
	"math/rand"
	"time"
)

// Do not use this. Instead, in your code, pass 0 as the TCP port and then retrieve the port for client usage:
// 		listener, err := net.Listen("tcp", "127.0.0.1:0")
//		port := listener.Addr().(*net.TCPAddr).Port
func RandomPort_UnsafeDoNotUseMe_I_Am_Going_Away() int {
	src := rand.NewSource(time.Now().UnixNano())
	return ((rand.New(src).Intn(25000) / 10) * 10) + 25111
}
