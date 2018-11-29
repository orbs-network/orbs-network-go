package test

import (
	"math/rand"
)

func RandomPort() int {

	return ((rand.Intn(25000) / 10) * 10) + 25111

	// our old implementation tried to find a "free" OS port but appears to be flaky
	//addr, _ := net.ResolveTCPAddr("tcp", "localhost:0")
	//l, _ := net.ListenTCP("tcp", addr)
	//defer l.Close()
	//return l.Addr().(*net.TCPAddr).Port
}
