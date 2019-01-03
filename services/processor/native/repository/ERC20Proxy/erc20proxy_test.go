package erc20proxy

import (
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBalance_AllGood(t *testing.T) {
	userHave := uint64(55)
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		state.WriteUint64(owner.RawAddress, userHave)
		// call
		balance := balanceOf(owner.RawAddress)
		require.Equal(t, userHave, balance)
	})
}

func TestBalance_WrongGoodAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		balance := balanceOf(owner.RawAddress)
		require.Equal(t, uint64(0), balance)
	})
}

func TestBalance_BadAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			balanceOf([]byte{0, 0, 4, 5})
		}, "should panic bad address")
	})
}

func TestTransfer_BadAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			transfer([]byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})

}

func TestTransferImpl_AllGood(t *testing.T) {
	userHave := uint64(50)
	targetHave := uint64(13)
	userTransfer := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, owner.RawAddress, func(m Mockery) {
		state.WriteUint64(owner.RawAddress, userHave)
		state.WriteUint64(target.RawAddress, targetHave)

		// call
		_transferImpl(owner.RawAddress, target.RawAddress, userTransfer)

		// assert
		require.Equal(t, userHave-userTransfer, state.ReadUint64(owner.RawAddress))
		require.Equal(t, targetHave+userTransfer, state.ReadUint64(target.RawAddress))
	})
}

func TestTransferImpl_NotEnough(t *testing.T) {
	userHave := uint64(12)
	targetHave := uint64(13)
	userTransfer := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		state.WriteUint64(owner.RawAddress, userHave)
		state.WriteUint64(target.RawAddress, targetHave)

		// call
		require.Panics(t, func() {
			_transferImpl(owner.RawAddress, target.RawAddress, userTransfer)
		}, "should panic not enough")
	})
}

func TestApproveAllow_AllGood(t *testing.T) {
	approveAmount := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	caller, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	spender, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, caller.RawAddress, func(m Mockery) {
		// call
		approve(spender.RawAddress, approveAmount)

		allowKey := append(caller.RawAddress, spender.RawAddress...)

		// assert
		require.Equal(t, approveAmount, state.ReadUint64(allowKey))
		require.Equal(t, approveAmount, allowance(caller.RawAddress, spender.RawAddress))
	})
}

func TestApprove_BadAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			approve([]byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

// TODO - rewrite once the sdk is better
/*func TestTransferFrom_AllGood(t *testing.T) {
	userHave := uint64(50)
	userTransfer := uint64(16)
	userApprove := uint64(20)

	from, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	spender, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")
	to, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")


	state.WriteUint64ByAddress(from.RawAddress, userHave)
	InServiceScope(nil, from.RawAddress, func(m Mockery) {
		approve(spender.RawAddress, userApprove)
	})

	InServiceScope(nil, spender.RawAddress, func(m Mockery) {
		// call
		transferFrom(from.RawAddress, to.RawAddress, userTransfer)
	})

	// assert
	require.Equal(t, userHave-userTransfer, state.ReadUint64ByAddress(from.RawAddress))
	require.Equal(t, userTransfer, state.ReadUint64ByAddress(to.RawAddress))
	require.Equal(t, userApprove-userTransfer, state.ReadUint64ByKey(_allowKey(from.RawAddress, spender.RawAddress)))
}

func TestTransferFrom_NotEnoughApprove(t *testing.T) {
	userHave := uint64(12)
	userTransfer := uint64(16)
	userApprove := uint64(13)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		state.WriteUint64ByAddress(owner.RawAddress, userHave)

		// call
		approve(target.RawAddress, userApprove)
		require.Panics(t, func() {
			transferFrom(owner.RawAddress, target.RawAddress, userTransfer)
		}, "should panic not enough")
	})
}
*/
func TestTransferFrom_BadSrcAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			transferFrom([]byte{0, 0, 4, 5}, owner.RawAddress, 10)
		}, "should panic bad address")
	})
}

func TestTransferFrom_BadTargetAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			transferFrom(owner.RawAddress, []byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

func TestMint(t *testing.T) {
	total := uint64(50)
	startWith := uint64(12)
	mintAmount := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, asbcontract.RawAddress, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target.RawAddress, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract.RawAddress)

		// call
		asbMint(target.RawAddress, mintAmount)

		// assert
		require.Equal(t, total+mintAmount, state.ReadUint64(TOTAL_SUPPLY_KEY))
		require.Equal(t, startWith+mintAmount, state.ReadUint64(target.RawAddress))
	})
}

func TestMint_BadAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, asbcontract.RawAddress, func(m Mockery) {
		state.WriteBytes(ASB_ADDR_KEY, asbcontract.RawAddress)
		// call
		require.Panics(t, func() {
			asbMint([]byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

func TestBurn_AllGood(t *testing.T) {
	total := uint64(50)
	startWith := uint64(22)
	burnAmount := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, asbcontract.RawAddress, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target.RawAddress, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract.RawAddress)

		// call
		asbBurn(target.RawAddress, burnAmount)

		// assert
		require.Equal(t, total-burnAmount, state.ReadUint64(TOTAL_SUPPLY_KEY))
		require.Equal(t, startWith-burnAmount, state.ReadUint64(target.RawAddress))
	})
}

func TestBurn_NotEnough(t *testing.T) {
	total := uint64(50)
	startWith := uint64(12)
	burnAmount := uint64(16)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.RawAddress, asbcontract.RawAddress, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target.RawAddress, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract.RawAddress)

		// call
		require.Panics(t, func() {
			asbBurn(target.RawAddress, burnAmount)
		}, "should panic not enough")
	})
}

func TestBurn_BadAddress(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, asbcontract.RawAddress, func(m Mockery) {
		state.WriteBytes(ASB_ADDR_KEY, asbcontract.RawAddress)
		// call
		require.Panics(t, func() {
			asbBurn([]byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

func TestBindAsb_AllGood(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, owner.RawAddress, func(m Mockery) {
		_init()

		// call
		asbBind(asbcontract.RawAddress)

		// assert
		require.Equal(t, asbcontract.RawAddress, state.ReadBytes(ASB_ADDR_KEY))
	})
}

func TestBindAsb_WrongCaller(t *testing.T) {
	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	caller, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	asbcontract, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")

	InServiceScope(owner.RawAddress, caller.RawAddress, func(m Mockery) {
		_init()

		// call
		require.Panics(t, func() {
			asbBind(asbcontract.RawAddress)
		}, "should panic bad caller")
	})
}
