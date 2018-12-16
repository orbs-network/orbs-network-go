package keys

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type ecdsaSecp256K1KeyPairHex struct {
	publicKey  string
	privateKey string
}

var ecdsaSecp256K1KeyPairs = []ecdsaSecp256K1KeyPairHex{
	{"0430fccea741dd34c7afb146a543616bcb361247148f0c8542541c01da6d6cadf186515f1d851978fc94a6a641e25dec74a6ec28c5ae04c651a0dc2e6104b3ac24", "901a1a0bfbe217593062a054e561e708707cb814a123474c25fd567a0fe088f8"},
	{"04e083ab26159e08171c8873295dde9859a9225fa9b50bb91deb972035406878647a18992f82d94930546ea255917ef9d31fd206db94e708db43e925e4efcdd1d3", "87a210586f57890ae3642c62ceb58f0f0a54e787891054a5a54c80e1da418253"},
	{"046a494cf5b39cbaea22615ab6da3f8d11b668d904849a2aadaac22b3e60487562e4ceb9c1328d598bb2c79eaff131c429e0f7c76717de429f6a7f385c627a2e48", "426308c4d11a6348a62b4fdfb30e2cad70ab039174e2e8ea707895e4c644c4ec"},
	{"040155dbfb5553c9e650078561344c5e23f35bf4639994e29c207a263aed766e57c21c35ab64ccd081e06a01220c13bd9219098c75d3d7bf4c096f382bdb7b457d", "1e404ba4e421cedf58dcc3dddcee656569afc7904e209612f7de93e1ad710300"},
	{"04a239a6f5fb4c0194623c0ea3ba89fb737922cde5f5c12bfcc19cf9ecee99b76dac32d24190b18b7cbf0d05155884342a182ed02ebebb8e975207e496957cf6e9", "0860f557af1b29639b680a5934e2080d204d08f753679e606f1bcb4b53d00efe"},
	{"04072ba860aa0710f21f97b2f74f8b7a7e6bac56bc5f273f0bea333159f352f0fe29b1e7c34ce3148c894e87323656f8127d64ae8dbebf2e48a7976629631ca6d7", "a8ca24ef5d3dc54df3a692ee5b27a9bfa06c4ae8ecf77e20db55acd7637087e1"},
	{"04ca3b4cf43a625dfb86d83acb78d40402995f17da1354810f51683176d159168bcfeb006b5e8b20a2c9f090a4aebfb8cd9cfa9783daad9f497ebc099be80aadf4", "a414d64a2246e394019f37544c17a8cae94b8f2104b9a5957c7af8691cb3302c"},
	{"04ecfc44abed95eb70feb664f753a07a41ba732ad78aa3fddd7d0a735068da9f3ec4265223ec36ab9d7de452630f3bdcace3a61d269a59c8c381e15ace66cb2d45", "97809939376f2cb7d0d0cdf6531b3389080f1342ed7ccc46d55aa3b0445fc906"},
	{"040c6a30b0f4de9bb6f35a7814c483f931b7185db2702a4af8aae79a3589a680f16046d1ba6bd3f40607e5a2ea6d2c44bd18b3303935eda96611a495d6ecc8e479", "f4cd604644c170c487643c22121270415356b6f791c32887b5d22b42bbf83505"},
	{"04a69b19f571824184f7847d9ed0e12f92b42688b56a28513d40683de09fb22f2071cbc80cc2850c435e7688b2a4534aa4b8f5deda0d65b2e3afe292ec260a0ac7", "c2ee571d81465cfcea039092a717b8460f8c96b82923b2ea7bb9765da8d013d6"},
}

func EcdsaSecp256K1KeyPairForTests(setIndex int) *keys.EcdsaSecp256K1KeyPair {
	if setIndex > len(ecdsaSecp256K1KeyPairs) {
		return nil
	}

	pub, err := hex.DecodeString(ecdsaSecp256K1KeyPairs[setIndex].publicKey)
	if err != nil {
		return nil
	}

	pri, err := hex.DecodeString(ecdsaSecp256K1KeyPairs[setIndex].privateKey)
	if err != nil {
		return nil
	}

	return keys.NewEcdsaSecp256K1KeyPair(pub, pri)
}

func EcdsaSecp256K1PublicKeysForTests() []primitives.EcdsaSecp256K1PublicKey {
	res := make([]primitives.EcdsaSecp256K1PublicKey, len(ecdsaSecp256K1KeyPairs))
	i := 0
	for _, pair := range ecdsaSecp256K1KeyPairs {
		res[i] = primitives.EcdsaSecp256K1PublicKey(pair.publicKey)
		i++
	}
	return res
}
