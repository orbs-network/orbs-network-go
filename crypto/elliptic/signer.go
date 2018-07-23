package elliptic

type Signer struct {
	publicKey        []byte
	privateKeyUnsafe []byte // in order to make this safe we need some sort of secure memory, there are external packages implementing this already
}

func NewPublicKeyString(pk string) *Signer {
	return nil
}

func NewPublicKeyBytes(pk []byte) *Signer {
	return nil
}

func (e *Signer) PublicKey() []byte {
	return e.publicKey
}

func (e *Signer) Sign(data []byte) string {
	// not implementing until we start using secure memory or other decision is made
	return ""
}

func (e *Signer) Verify(data []byte, sig string) bool {
	return false
}
