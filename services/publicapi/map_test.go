package publicapi

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestPublicApiWaiter_AddToMap(t *testing.T) {
	key := "random"
	mapper := newMapper()
	wc, err := mapper.addWaiter(key)

	require.NoError(t, err, "error happened when it should not")
	require.NotNil(t, wc, "wait object is nil when it should exist")
}

func TestPublicApiWaiter_AddToMapTwice(t *testing.T) {
	key := "random"
	mapper := newMapper()
	_, err := mapper.addWaiter(key)
	wc, err := mapper.addWaiter(key)

	require.Error(t, err, "error did not happened when it should")
	require.Nil(t, wc, "wait object is not nil when it should")
}

// TODO MUTEX test

func TestPublicApiWaiter_DeleteFromMap(t *testing.T) {
	key := "random"
	mapper := newMapper()
	mapper.addWaiter(key)
	wc, err := mapper.delete(key)

	require.NoError(t, err, "error happened when it should not")
	require.NotNil(t, wc, "wait object is nil when it should not")
}

func TestPublicApiWaiter_DeleteFromMapTwice(t *testing.T) {
	key := "random"
	mapper := newMapper()
	mapper.addWaiter(key)
	mapper.delete(key)
	wc, err := mapper.delete(key)

	require.Error(t, err, "error did not happened when it should")
	require.Nil(t, wc, "wait object is not nil when it should")
}

// TODO MUTEX TEST

