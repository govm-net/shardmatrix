package crypto

import (
	"errors"
	"io"

	_ "github.com/govm-net/shardmatrix/pkg/crypto/ed25519"
	_ "github.com/govm-net/shardmatrix/pkg/crypto/schnorr"
	_ "github.com/govm-net/shardmatrix/pkg/crypto/secp256k1"
	"github.com/govm-net/shardmatrix/pkg/crypto/types"
)

// NewSigner creates a new signer based on the specified algorithm
func NewSigner(alg types.Algorithm) (types.Signer, error) {
	factory, err := types.GetSignerFactory(alg)
	if err != nil {
		return nil, err
	}
	return factory(), nil
}

// NewVerifier creates a new verifier based on the specified algorithm
func NewVerifier(alg types.Algorithm) (types.Verifier, error) {
	factory, err := types.GetVerifierFactory(alg)
	if err != nil {
		return nil, err
	}
	return factory(), nil
}

// GenerateKey generates a new key pair for the specified algorithm
func GenerateKey(alg types.Algorithm, rand io.Reader) (types.Signer, error) {
	signer, err := NewSigner(alg)
	if err != nil {
		return nil, err
	}

	if err := signer.GenerateKey(rand); err != nil {
		return nil, err
	}

	return signer, nil
}

// ImportPrivateKey imports a private key for the specified algorithm
func ImportPrivateKey(alg types.Algorithm, privKey []byte) (types.Signer, error) {
	signer, err := NewSigner(alg)
	if err != nil {
		return nil, err
	}

	if err := signer.ImportPrivateKey(privKey); err != nil {
		return nil, err
	}

	return signer, nil
}

// ImportPublicKey imports a public key for the specified algorithm
func ImportPublicKey(alg types.Algorithm, pubKey []byte) (types.Verifier, error) {
	verifier, err := NewVerifier(alg)
	if err != nil {
		return nil, err
	}

	if err := verifier.ImportPublicKey(pubKey); err != nil {
		return nil, err
	}

	return verifier, nil
}

var (
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)
