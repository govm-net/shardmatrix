package ed25519

import (
	"crypto"
	"crypto/ed25519"
	"errors"
	"io"

	"github.com/govm-net/shardmatrix/pkg/crypto/types"
)

type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func init() {
	types.RegisterSigner(types.Ed25519, func() types.Signer {
		return NewSigner()
	})
	types.RegisterVerifier(types.Ed25519, func() types.Verifier {
		return NewSigner()
	})
}

func NewSigner() *Signer {
	return &Signer{}
}

func (s *Signer) Type() types.Algorithm {
	return types.Ed25519
}

func (s *Signer) Sign(msg []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, errors.New("private key not set")
	}
	return ed25519.Sign(s.privateKey, msg), nil
}

func (s *Signer) Verify(msg, sig []byte) bool {
	return ed25519.Verify(s.publicKey, msg, sig)
}

func (s *Signer) PublicKey() crypto.PublicKey {
	return s.publicKey
}

func (s *Signer) PrivateKey() crypto.PrivateKey {
	return s.privateKey
}

func (s *Signer) GenerateKey(rand io.Reader) error {
	pub, priv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return err
	}
	s.privateKey = priv
	s.publicKey = pub
	return nil
}

func (s *Signer) ImportPrivateKey(privKey []byte) error {
	if len(privKey) != ed25519.PrivateKeySize {
		return errors.New("invalid private key length")
	}
	s.privateKey = privKey
	s.publicKey = ed25519.PrivateKey(privKey).Public().(ed25519.PublicKey)
	return nil
}

func (s *Signer) ImportPublicKey(pubKey []byte) error {
	s.publicKey = ed25519.PublicKey(pubKey)
	return nil
}

func (s *Signer) RecoverPublicKey(msg, sig []byte) (crypto.PublicKey, error) {
	return nil, errors.New("ed25519 signer does not support recovering public key")
}
