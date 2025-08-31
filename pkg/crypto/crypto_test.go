package crypto

import (
	"bytes"
	"testing"

	"github.com/govm-net/shardmatrix/pkg/types"
)

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if keyPair.PrivateKey == nil {
		t.Error("Private key should not be nil")
	}

	if keyPair.PublicKey == nil {
		t.Error("Public key should not be nil")
	}

	// 验证地址生成
	address := keyPair.GetAddress()
	if address.IsZero() {
		t.Error("Generated address should not be zero")
	}
}

func TestSignAndVerify(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	testData := []byte("test message for signing")

	// 测试签名
	signature, err := keyPair.SignData(testData)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	if len(signature) == 0 {
		t.Error("Signature should not be empty")
	}

	// 测试验证
	isValid := keyPair.VerifySignature(testData, signature)
	if !isValid {
		t.Error("Signature verification should succeed")
	}

	// 测试错误数据验证
	wrongData := []byte("wrong message")
	isValid = keyPair.VerifySignature(wrongData, signature)
	if isValid {
		t.Error("Signature verification should fail for wrong data")
	}

	// 测试错误签名验证
	wrongSignature := make([]byte, len(signature))
	copy(wrongSignature, signature)
	wrongSignature[0] ^= 0xFF // 修改第一个字节
	isValid = keyPair.VerifySignature(testData, wrongSignature)
	if isValid {
		t.Error("Signature verification should fail for wrong signature")
	}
}

func TestPublicKeyToAddress(t *testing.T) {
	keyPair1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}

	keyPair2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	address1 := PublicKeyToAddress(keyPair1.PublicKey)
	address2 := PublicKeyToAddress(keyPair2.PublicKey)

	// 不同的公钥应该生成不同的地址
	if bytes.Equal(address1[:], address2[:]) {
		t.Error("Different public keys should generate different addresses")
	}

	// 相同的公钥应该生成相同的地址
	address1Copy := PublicKeyToAddress(keyPair1.PublicKey)
	if !bytes.Equal(address1[:], address1Copy[:]) {
		t.Error("Same public key should generate same address")
	}
}

func TestHashData(t *testing.T) {
	testData := []byte("test data for hashing")

	hash1 := HashData(testData)
	hash2 := HashData(testData)

	// 相同数据应该产生相同哈希
	if hash1 != hash2 {
		t.Error("Same data should produce same hash")
	}

	// 不同数据应该产生不同哈希
	differentData := []byte("different test data")
	hash3 := HashData(differentData)
	if hash1 == hash3 {
		t.Error("Different data should produce different hash")
	}

	// 验证哈希长度
	if len(hash1.Bytes()) != 32 {
		t.Errorf("Hash should be 32 bytes, got %d", len(hash1.Bytes()))
	}
}

func TestValidateAddress(t *testing.T) {
	// 测试有效地址
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	validAddress := keyPair.GetAddress()
	if !ValidateAddress(validAddress) {
		t.Error("Valid address should pass validation")
	}

	// 测试零地址
	var zeroAddress types.Address
	if ValidateAddress(zeroAddress) {
		t.Error("Zero address should fail validation")
	}
}

func TestSigner(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signer := NewSigner(keyPair)
	testData := []byte("test message for signer")

	// 测试签名
	signature, err := signer.Sign(testData)
	if err != nil {
		t.Fatalf("Failed to sign with signer: %v", err)
	}

	// 测试公钥获取
	publicKey := signer.GetPublicKey()
	if publicKey == nil {
		t.Error("Public key should not be nil")
	}

	// 测试地址获取
	address := signer.GetAddress()
	if address.IsZero() {
		t.Error("Address should not be zero")
	}

	// 验证签名
	verifier := NewVerifier()
	isValid := verifier.Verify(testData, signature, publicKey)
	if !isValid {
		t.Error("Signature verification should succeed")
	}
}

func TestPublicKeyFromBytes(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// 将公钥转换为字节
	pubKeyBytes := append(keyPair.PublicKey.X.Bytes(), keyPair.PublicKey.Y.Bytes()...)

	// 确保字节长度为64
	if len(pubKeyBytes) < 64 {
		padding := make([]byte, 64-len(pubKeyBytes))
		pubKeyBytes = append(padding, pubKeyBytes...)
	}
	if len(pubKeyBytes) > 64 {
		pubKeyBytes = pubKeyBytes[len(pubKeyBytes)-64:]
	}

	// 从字节恢复公钥
	recoveredPubKey, err := PublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		t.Fatalf("Failed to recover public key from bytes: %v", err)
	}

	// 验证恢复的公钥是否正确
	originalAddress := PublicKeyToAddress(keyPair.PublicKey)
	recoveredAddress := PublicKeyToAddress(recoveredPubKey)

	if originalAddress != recoveredAddress {
		t.Error("Recovered public key should generate same address")
	}

	// 测试无效长度
	invalidBytes := []byte{0x01, 0x02, 0x03}
	_, err = PublicKeyFromBytes(invalidBytes)
	if err == nil {
		t.Error("Should fail for invalid byte length")
	}
}

func TestIntegrationSigningFlow(t *testing.T) {
	// 创建发送者和接收者
	senderKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate sender key pair: %v", err)
	}

	receiverKeyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate receiver key pair: %v", err)
	}

	// 创建交易
	from := senderKeyPair.GetAddress()
	to := receiverKeyPair.GetAddress()
	tx := types.NewTransaction(from, to, 100, 10, 1, []byte("test transaction"))

	// 计算交易哈希
	txHash := tx.Hash()

	// 发送者签名交易
	signature, err := senderKeyPair.SignData(txHash.Bytes())
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 设置签名
	tx.Signature = signature

	// 验证签名
	isValid := VerifySignature(txHash.Bytes(), signature, senderKeyPair.PublicKey)
	if !isValid {
		t.Error("Transaction signature verification should succeed")
	}

	// 验证使用错误公钥的签名应该失败
	isValid = VerifySignature(txHash.Bytes(), signature, receiverKeyPair.PublicKey)
	if isValid {
		t.Error("Transaction signature verification should fail with wrong public key")
	}
}
