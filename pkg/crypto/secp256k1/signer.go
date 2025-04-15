package secp256k1

import (
	"crypto"
	"errors"
	"io"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/govm-net/shardmatrix/pkg/crypto/types"
)

type Signer struct {
	privateKey *btcec.PrivateKey
}

type Verifier struct {
	publicKey *btcec.PublicKey
}

func init() {
	types.RegisterSigner(types.Secp256k1, func() types.Signer {
		return NewSigner()
	})
	types.RegisterVerifier(types.Secp256k1, func() types.Verifier {
		return NewVerifier()
	})
}

func NewSigner() *Signer {
	return &Signer{}
}

func NewVerifier() *Verifier {
	return &Verifier{}
}

func (s *Signer) Type() types.Algorithm {
	return types.Secp256k1
}

func (v *Verifier) Type() types.Algorithm {
	return types.Secp256k1
}

func (s *Signer) Sign(msg []byte) ([]byte, error) {
	if s.privateKey == nil {
		return nil, errors.New("private key not set")
	}
	hash := types.GetHash(msg)
	sig := ecdsa.SignCompact(s.privateKey, hash, true)
	return sig, nil
}

func (s *Signer) Verify(msg, sig []byte) bool {
	if s.privateKey == nil {
		return false
	}
	hash := types.GetHash(msg)
	pubKey, _, err := ecdsa.RecoverCompact(sig, hash)
	if err != nil {
		return false
	}
	return pubKey.IsEqual(s.privateKey.PubKey())
}

func (s *Signer) PublicKey() crypto.PublicKey {
	if s.privateKey == nil {
		return nil
	}
	return s.privateKey.PubKey()
}

func (s *Signer) PrivateKey() crypto.PrivateKey {
	return s.privateKey
}

func (s *Signer) GenerateKey(rand io.Reader) error {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return err
	}
	s.privateKey = privKey
	return nil
}

func (s *Signer) ImportPrivateKey(privKey []byte) error {
	priv, _ := btcec.PrivKeyFromBytes(privKey)
	s.privateKey = priv
	return nil
}

func (s *Signer) ImportPublicKey(pubKey []byte) error {
	return errors.New("secp256k1 signer does not support importing public key")
}

func (v *Verifier) Verify(msg, sig []byte) bool {
	if v.publicKey == nil {
		return false
	}
	hash := types.GetHash(msg)
	pubKey, _, err := ecdsa.RecoverCompact(sig, hash)
	if err != nil {
		return false
	}
	return pubKey.IsEqual(v.publicKey)
}

func (v *Verifier) PublicKey() crypto.PublicKey {
	return v.publicKey
}

func (v *Verifier) ImportPublicKey(pubKey []byte) error {
	pub, err := btcec.ParsePubKey(pubKey)
	if err != nil {
		return err
	}
	v.publicKey = pub
	return nil
}

func (v *Verifier) RecoverPublicKey(msg, sig []byte) (crypto.PublicKey, error) {
	hash := types.GetHash(msg)
	pubKey, _, err := ecdsa.RecoverCompact(sig, hash)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}
