package secp256k1

import (
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestSecp256k1Signer(t *testing.T) {
	// 创建签名器
	signer := NewSigner()

	// 生成密钥对
	if err := signer.GenerateKey(rand.Reader); err != nil {
		t.Fatal(err)
	}

	// 测试签名
	msg := []byte("hello world")
	sig, err := signer.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}

	// 测试验签
	valid := signer.Verify(msg, sig)
	if !valid {
		t.Fatal("signature verification failed")
	}

	// 测试导入私钥
	privKey := signer.PrivateKey()
	newSigner := NewSigner()
	if err := newSigner.ImportPrivateKey(privKey.(*btcec.PrivateKey).Serialize()); err != nil {
		t.Fatal(err)
	}

	// 验证导入的私钥可以正确签名
	sig2, err := newSigner.Sign(msg)
	if err != nil {
		t.Fatal(err)
	}
	valid = newSigner.Verify(msg, sig2)
	if !valid {
		t.Fatal("signature verification failed with imported private key")
	}

	// 测试导入公钥到验证器
	verifier := NewVerifier()
	if err := verifier.ImportPublicKey(signer.PublicKey().(*btcec.PublicKey).SerializeCompressed()); err != nil {
		t.Fatal(err)
	}

	// 验证导入的公钥可以正确验签
	valid = verifier.Verify(msg, sig)
	if !valid {
		t.Fatal("signature verification failed with imported public key")
	}

	// 测试从签名恢复公钥
	recoveredPubKey, err := verifier.RecoverPublicKey(msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	// 验证恢复的公钥与原始公钥相同
	originalPubKey := signer.PublicKey().(*btcec.PublicKey)
	recoveredPubKeyBytes := recoveredPubKey.(*btcec.PublicKey).SerializeCompressed()
	originalPubKeyBytes := originalPubKey.SerializeCompressed()
	if string(recoveredPubKeyBytes) != string(originalPubKeyBytes) {
		t.Fatal("recovered public key does not match original public key")
	}

	// 使用恢复的公钥验证签名
	newVerifier := NewVerifier()
	if err := newVerifier.ImportPublicKey(recoveredPubKeyBytes); err != nil {
		t.Fatal(err)
	}
	valid = newVerifier.Verify(msg, sig)
	if !valid {
		t.Fatal("signature verification failed with recovered public key")
	}

	// 测试不支持的功能
	err = signer.ImportPublicKey([]byte("test"))
	if err == nil {
		t.Fatal("expected error when importing public key to signer")
	}
}
