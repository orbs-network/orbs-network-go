package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestLeaderCreatesBlocks(t *testing.T) {
	c := newContext(true)
	c.consensusContext.When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).AtLeast(1)
	c.createService()
	err := test.EventuallyVerify(c.consensusContext)
	if err != nil {
		t.Fatal("Did not create block with ConsensusContext:", err)
	}
}

func TestNonLeaderDoesNotCreateBlocks(t *testing.T) {
	c := newContext(false)
	c.consensusContext.When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).Times(0)
	c.createService()
	err := test.ConsistentlyVerify(c.consensusContext)
	if err != nil {
		t.Fatal("Did create block with ConsensusContext:", err)
	}
}
