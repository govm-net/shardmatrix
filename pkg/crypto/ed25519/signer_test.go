package ed25519

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestEd25519Signer(t *testing.T) {
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
	if err := newSigner.ImportPrivateKey(privKey.(ed25519.PrivateKey)); err != nil {
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
	verifier := NewSigner()
	pubKey := signer.PublicKey().(ed25519.PublicKey)
	if err := verifier.ImportPublicKey(pubKey); err != nil {
		t.Fatal(err)
	}

	// 验证导入的公钥可以正确验签
	valid = verifier.Verify(msg, sig)
	if !valid {
		t.Fatal("signature verification failed with imported public key")
	}
}
