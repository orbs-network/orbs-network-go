package acceptance

import (
	"fmt"
	"testing"
	"github.com/orbs-network/lean-helix-go/go/leanhelix"
)

// This is just to allow importing lean-helix-go as submodule (it must be used someplace).
// Once that lib is used in real tests, can delete this file
func TestLeanHelixNoOp(t *testing.T) {

	s := leanhelix.NewLeanHelix()
	fmt.Println(s)

}
