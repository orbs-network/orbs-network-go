package asb_ether

import (
	"github.com/orbs-network/orbs-client-sdk-go/orbsclient"
	. "github.com/orbs-network/orbs-contract-sdk/go/fake"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/safemath/safeuint64"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
)

func TestTransferIn_AllGood(t *testing.T) {
	txid := "cccc"

	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.RawAddress)

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = big.NewInt(42)
			v.To = orbsUserAddress
			v.Value = big.NewInt(17)

		})

		// this is what we expect to be called
		m.MockServiceCallMethod(getTokenContract(), "mint", nil, orbsUser.RawAddress, uint64(17))

		// call
		transferIn(txid)

		// assert
		m.VerifyMocks()
		require.True(t, isInTuidExists(genInTuidKey(42)))
	})

}

func TestTransferIn_NoTuid(t *testing.T) {
	txid := "cccc"

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = nil
		})

		// call
		require.Panics(t, func() {
			transferIn(txid)
		}, "should panic because no tuid")
	})
}

func TestTransferIn_NoValue(t *testing.T) {
	txid := "cccc"

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = big.NewInt(42)
		})

		// call
		require.Panics(t, func() {
			transferIn(txid)
		}, "should panic because no value")
	})
}

func TestTransferIn_NegativeValue(t *testing.T) {
	txid := "cccc"

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = big.NewInt(42)
			v.Value = big.NewInt(-17)
		})

		// call
		require.Panics(t, func() {
			transferIn(txid)
		}, "should panic because negative value")
	})
}

func TestTransferIn_NoOrbsAddress(t *testing.T) {
	txid := "cccc"

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = big.NewInt(42)
			v.Value = big.NewInt(17)
		})

		// call
		require.Panics(t, func() {
			transferIn(txid)
		}, "should panic because no address")
	})
}

func TestTransferIn_TuidAlreadyUsed(t *testing.T) {
	txid := "cccc"

	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.RawAddress)

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug
		setInTuid(genInTuidKey(42))

		// prepare
		m.MockEthereumLog(getAsbAddr(), getAsbAbi(), txid, "EthTransferredOut", func(out interface{}) {
			v := out.(*EthTransferredOut)
			v.Tuid = big.NewInt(42)
			v.Value = big.NewInt(17)
			v.To = orbsUserAddress
		})

		// call
		require.Panics(t, func() {
			transferIn(txid)
		}, "should panic because no address")
	})
}

func TestTransferOut_AllGood(t *testing.T) {
	amount := uint64(17)
	ethAddr := AnAddress()

	orbsUser, err := orbsclient.CreateAccount()
	require.NoError(t, err, "could not create orbs address")
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.RawAddress)

	InServiceScope(orbsUser.RawAddress, nil, func(m Mockery) {
		_init() // start the asb contracat // todo  v1 open bug

		// what is expected to be called
		tuid := safeuint64.Add(getOutTuid(), 1)
		m.MockEmitEvent(OrbsTransferredOut, tuid, orbsUser.RawAddress, ethAddr, big.NewInt(17).Uint64())
		m.MockServiceCallMethod(getTokenContract(), "burn", nil, orbsUser.RawAddress, amount)

		// call
		transferOut(ethAddr, amount)

		// assert
		m.VerifyMocks()
		require.Equal(t, uint64(1), getOutTuid())
	})
}

func TestReset(t *testing.T) {
	maxOut := uint64(500)
	maxIn := uint64(200)

	InServiceScope(nil, nil, func(m Mockery) {
		_init() // start the asb contracat

		setOutTuid(maxOut)
		for i := uint64(0); i < maxIn; i++ {
			if i%54 == 0 {
				continue // just as not to have all of them
			}
			setInTuid(genInTuidKey(i))
		}
		setInTuidMax(maxIn)

		// call
		resetContract()

		// assert
		require.Equal(t, uint64(0), getOutTuid())
		require.Equal(t, uint64(0), getInTuidMax())
		for i := uint64(0); i < maxIn; i++ {
			require.False(t, isInTuidExists(genInTuidKey(i)), "tuid should be empty %d", i)
		}
	})
}
