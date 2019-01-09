package erc20proxy

import (
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBalance_AllGood(t *testing.T) {
	userHave := uint64(55)
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		state.WriteUint64(owner, userHave)
		// call
		balance := balanceOf(owner)
		require.Equal(t, userHave, balance)
	})
}

func TestBalance_WrongGoodAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		// call
		balance := balanceOf(owner)
		require.Equal(t, uint64(0), balance)
	})
}

func TestBalance_BadAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			balanceOf([]byte{0, 0, 4, 5})
		}, "should panic bad address")
	})
}

func TestTransfer_BadAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
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

	owner := createOrbsAddress()
	target := createOrbsAddress()

	InServiceScope(owner, owner, func(m Mockery) {
		state.WriteUint64(owner, userHave)
		state.WriteUint64(target, targetHave)

		// call
		_transferImpl(owner, target, userTransfer)

		// assert
		require.Equal(t, userHave-userTransfer, state.ReadUint64(owner))
		require.Equal(t, targetHave+userTransfer, state.ReadUint64(target))
	})
}

func TestTransferImpl_NotEnough(t *testing.T) {
	userHave := uint64(12)
	targetHave := uint64(13)
	userTransfer := uint64(16)

	owner := createOrbsAddress()
	target := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		state.WriteUint64(owner, userHave)
		state.WriteUint64(target, targetHave)

		// call
		require.Panics(t, func() {
			_transferImpl(owner, target, userTransfer)
		}, "should panic not enough")
	})
}

func TestApproveAllow_AllGood(t *testing.T) {
	approveAmount := uint64(16)

	owner := createOrbsAddress()
	caller := createOrbsAddress()
	spender := createOrbsAddress()

	InServiceScope(owner, caller, func(m Mockery) {
		// call
		approve(spender, approveAmount)

		allowKey := append(caller, spender...)

		// assert
		require.Equal(t, approveAmount, state.ReadUint64(allowKey))
		require.Equal(t, approveAmount, allowance(caller, spender))
	})
}

func TestApprove_BadAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
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


	state.WriteUint64ByAddress(from.AddressAsBytes(), userHave)
	InServiceScope(nil, from.AddressAsBytes(), func(m Mockery) {
		approve(spender.AddressAsBytes(), userApprove)
	})

	InServiceScope(nil, spender.AddressAsBytes(), func(m Mockery) {
		// call
		transferFrom(from.AddressAsBytes(), to.AddressAsBytes(), userTransfer)
	})

	// assert
	require.Equal(t, userHave-userTransfer, state.ReadUint64ByAddress(from.AddressAsBytes()))
	require.Equal(t, userTransfer, state.ReadUint64ByAddress(to.AddressAsBytes()))
	require.Equal(t, userApprove-userTransfer, state.ReadUint64ByKey(_allowKey(from.AddressAsBytes(), spender.AddressAsBytes())))
}

func TestTransferFrom_NotEnoughApprove(t *testing.T) {
	userHave := uint64(12)
	userTransfer := uint64(16)
	userApprove := uint64(13)

	owner, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 1")
	target, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address 2")

	InServiceScope(owner.AddressAsBytes(), nil, func(m Mockery) {
		state.WriteUint64ByAddress(owner.AddressAsBytes(), userHave)

		// call
		approve(target.AddressAsBytes(), userApprove)
		require.Panics(t, func() {
			transferFrom(owner.AddressAsBytes(), target.AddressAsBytes(), userTransfer)
		}, "should panic not enough")
	})
}
*/
func TestTransferFrom_BadSrcAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			transferFrom([]byte{0, 0, 4, 5}, owner, 10)
		}, "should panic bad address")
	})
}

func TestTransferFrom_BadTargetAddress(t *testing.T) {
	owner := createOrbsAddress()

	InServiceScope(owner, nil, func(m Mockery) {
		// call
		require.Panics(t, func() {
			transferFrom(owner, []byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

func TestMint(t *testing.T) {
	total := uint64(50)
	startWith := uint64(12)
	mintAmount := uint64(16)

	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()
	target := createOrbsAddress()

	InServiceScope(owner, asbcontract, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract)

		// call
		asbMint(target, mintAmount)

		// assert
		require.Equal(t, total+mintAmount, state.ReadUint64(TOTAL_SUPPLY_KEY))
		require.Equal(t, startWith+mintAmount, state.ReadUint64(target))
	})
}

func TestMint_BadAddress(t *testing.T) {
	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()

	InServiceScope(owner, asbcontract, func(m Mockery) {
		state.WriteBytes(ASB_ADDR_KEY, asbcontract)
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

	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()
	target := createOrbsAddress()

	InServiceScope(owner, asbcontract, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract)

		// call
		asbBurn(target, burnAmount)

		// assert
		require.Equal(t, total-burnAmount, state.ReadUint64(TOTAL_SUPPLY_KEY))
		require.Equal(t, startWith-burnAmount, state.ReadUint64(target))
	})
}

func TestBurn_NotEnough(t *testing.T) {
	total := uint64(50)
	startWith := uint64(12)
	burnAmount := uint64(16)

	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()
	target := createOrbsAddress()

	InServiceScope(owner, asbcontract, func(m Mockery) {
		state.WriteUint64(TOTAL_SUPPLY_KEY, total)
		state.WriteUint64(target, startWith)
		state.WriteBytes(ASB_ADDR_KEY, asbcontract)

		// call
		require.Panics(t, func() {
			asbBurn(target, burnAmount)
		}, "should panic not enough")
	})
}

func TestBurn_BadAddress(t *testing.T) {
	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()

	InServiceScope(owner, asbcontract, func(m Mockery) {
		state.WriteBytes(ASB_ADDR_KEY, asbcontract)
		// call
		require.Panics(t, func() {
			asbBurn([]byte{0, 0, 4, 5}, 10)
		}, "should panic bad address")
	})
}

func TestBindAsb_AllGood(t *testing.T) {
	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()

	InServiceScope(owner, owner, func(m Mockery) {
		_init()

		// call
		asbBind(asbcontract)

		// assert
		require.Equal(t, asbcontract, state.ReadBytes(ASB_ADDR_KEY))
	})
}

func TestBindAsb_WrongCaller(t *testing.T) {
	owner := createOrbsAddress()
	asbcontract := createOrbsAddress()
	caller := createOrbsAddress()

	InServiceScope(owner, caller, func(m Mockery) {
		_init()

		// call
		require.Panics(t, func() {
			asbBind(asbcontract)
		}, "should panic bad caller")
	})
}

// TODO(v1): talkol - I will move this to be part of the test framework
func createOrbsAddress() []byte {
	orbsUser, err := orbsClient.CreateAccount()
	if err != nil {
		panic(err.Error())
	}
	return orbsUser.AddressAsBytes()
}
