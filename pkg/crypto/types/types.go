package types

import (
	"crypto"
	"errors"
	"io"
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Signer defines the interface for signing operations
type Signer interface {
	// Type returns the algorithm type
	Type() Algorithm

	// Sign signs the given message
	Sign(msg []byte) ([]byte, error)

	// Verify verifies the signature against the message
	Verify(msg, sig []byte) bool

	// PublicKey returns the public key associated with the signer
	PublicKey() crypto.PublicKey

	// PrivateKey returns the private key associated with the signer
	PrivateKey() crypto.PrivateKey

	// GenerateKey generates a new key pair
	GenerateKey(rand io.Reader) error

	// ImportPrivateKey imports a private key
	ImportPrivateKey(privKey []byte) error
}

// Verifier defines the interface for verification operations
type Verifier interface {
	// Type returns the algorithm type
	Type() Algorithm

	// Verify verifies the signature against the message
	Verify(msg, sig []byte) bool

	// PublicKey returns the public key
	PublicKey() crypto.PublicKey

	// ImportPublicKey imports a public key
	ImportPublicKey(pubKey []byte) error

	// RecoverPublicKey recovers the public key from the signature and message
	RecoverPublicKey(msg, sig []byte) (crypto.PublicKey, error)
}

// Algorithm 定义支持的签名算法
type Algorithm byte

const (
	// Secp256k1 使用 secp256k1 曲线的 ECDSA 签名算法
	Secp256k1 Algorithm = iota + 1
	// Schnorr 使用 secp256k1 曲线的 Schnorr 签名算法
	Schnorr
	// Ed25519 使用 Ed25519 签名算法
	Ed25519
)

// SignerFactory creates a new signer
type SignerFactory func() Signer

// VerifierFactory creates a new verifier
type VerifierFactory func() Verifier

var (
	signerFactories   = make(map[Algorithm]SignerFactory)
	verifierFactories = make(map[Algorithm]VerifierFactory)
	mu                sync.RWMutex
)

// RegisterSigner registers a signer factory for the given algorithm
func RegisterSigner(alg Algorithm, factory SignerFactory) {
	mu.Lock()
	defer mu.Unlock()
	signerFactories[alg] = factory
}

// RegisterVerifier registers a verifier factory for the given algorithm
func RegisterVerifier(alg Algorithm, factory VerifierFactory) {
	mu.Lock()
	defer mu.Unlock()
	verifierFactories[alg] = factory
}

// GetSignerFactory returns the signer factory for the given algorithm
func GetSignerFactory(alg Algorithm) (SignerFactory, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := signerFactories[alg]
	if !ok {
		return nil, ErrUnsupportedAlgorithm
	}
	return factory, nil
}

// GetVerifierFactory returns the verifier factory for the given algorithm
func GetVerifierFactory(alg Algorithm) (VerifierFactory, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := verifierFactories[alg]
	if !ok {
		return nil, ErrUnsupportedAlgorithm
	}
	return factory, nil
}

var (
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)

type HashFunc func(data []byte) []byte

var GetHash HashFunc = func(data []byte) []byte {
	return chainhash.DoubleHashB(data)
}
