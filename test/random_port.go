package test

import (
	"math/rand"
	"time"
)

func RandomPort() int {
	src := rand.NewSource(time.Now().UnixNano())
	return ((rand.New(src).Intn(25000) / 10) * 10) + 25111

	// our old implementation tried to find a "free" OS port but appears to be flaky
	//addr, _ := net.ResolveTCPAddr("tcp", "localhost:0")
	//l, _ := net.ListenTCP("tcp", addr)
	//defer l.Close()
	//return l.Addr().(*net.TCPAddr).Port
}
