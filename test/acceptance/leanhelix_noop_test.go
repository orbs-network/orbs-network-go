package acceptance

import (
	"github.com/orbs-network/lean-helix-go/go/leanhelix"
	"testing"
)

// This is just to allow importing lean-helix-go as submodule (it must be used someplace).
// Once that lib is used in real tests, can delete this file

func TestLeanHelixNoOp(t *testing.T) {

	leanhelix.NewLeanHelix()

}
