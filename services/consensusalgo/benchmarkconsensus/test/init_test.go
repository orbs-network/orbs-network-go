package test

import (
	"testing"
)

func TestInit(t *testing.T) {
	c := newContext(true)
	c.createService()
	ok, err := c.gossip.Verify()
	if !ok {
		t.Fatal("Did not register with Gossip:", err)
	}
	ok, err = c.blockStorage.Verify()
	if !ok {
		t.Fatal("Did not register with BlockStorage:", err)
	}
}
