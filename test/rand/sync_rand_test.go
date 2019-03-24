// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package rand

import (
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func TestSyncRandWorksLikeRand(t *testing.T) {
	seed := int64(17)
	gorand := rand.New(rand.NewSource(seed))
	syncrand := newSyncRand(seed)

	// set both objects with an arbitrary new seed
	newSeed := time.Now().UnixNano()
	gorand.Seed(newSeed)
	syncrand.Seed(newSeed)

	goshuffarr := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	syncshuffarr := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	gorand.Shuffle(10, func(i, j int) {
		temp := goshuffarr[i]
		goshuffarr[i] = goshuffarr[j]
		goshuffarr[j] = temp
	})
	syncrand.Shuffle(10, func(i, j int) {
		temp := syncshuffarr[i]
		syncshuffarr[i] = syncshuffarr[j]
		syncshuffarr[j] = temp
	})
	require.EqualValues(t, syncshuffarr, goshuffarr, "expected shuffled arrays to match")
	syncArrLen, _ := syncrand.Read(syncshuffarr)
	goArrLen, _ := gorand.Read(goshuffarr)
	require.EqualValues(t, syncshuffarr, goshuffarr, "expected read arrays to match")
	require.EqualValues(t, goArrLen, syncArrLen)
	require.EqualValues(t, gorand.Perm(10), syncrand.Perm(10), "expected permutations to match")

	require.EqualValues(t, gorand.Float32(), syncrand.Float32())
	require.EqualValues(t, gorand.Float64(), syncrand.Float64())

	require.EqualValues(t, gorand.Int(), syncrand.Int())
	require.EqualValues(t, gorand.Int31(), syncrand.Int31())
	require.EqualValues(t, gorand.Int63(), syncrand.Int63())

	require.EqualValues(t, gorand.Intn(100), syncrand.Intn(100))
	require.EqualValues(t, gorand.Int31n(100), syncrand.Int31n(100))
	require.EqualValues(t, gorand.Int63n(100), syncrand.Int63n(100))

	require.EqualValues(t, gorand.Uint32(), syncrand.Uint32())
	require.EqualValues(t, gorand.Uint64(), syncrand.Uint64())
}
