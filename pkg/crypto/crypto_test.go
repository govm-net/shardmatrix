package crypto

import (
	"testing"
	"time"
)

// TestBasicCrypto 测试基础加密功能
func TestBasicCrypto(t *testing.T) {
	t.Run("KeyPairGeneration", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}
		
		if keyPair.Address.IsEmpty() {
			t.Error("Generated address should not be empty")
		}
		
		if len(keyPair.PublicKey) == 0 {
			t.Error("Public key should not be empty")
		}
	})
	
	t.Run("SignatureVerification", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}
		
		testData := []byte("ShardMatrix test data")
		signature := keyPair.Sign(testData)
		
		if signature.IsEmpty() {
			t.Error("Signature should not be empty")
		}
		
		if !VerifySignature(keyPair.PublicKey, testData, signature) {
			t.Error("Signature verification failed")
		}
		
		// Test with wrong data
		wrongData := []byte("Wrong data")
		if VerifySignature(keyPair.PublicKey, wrongData, signature) {
			t.Error("Should not verify signature for wrong data")
		}
	})
	
	t.Run("AddressGeneration", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}
		
		address1 := PublicKeyToAddress(keyPair.PublicKey)
		address2 := keyPair.Address
		
		if address1 != address2 {
			t.Error("Address generation inconsistency")
		}
	})
	
	t.Run("HashDeterminism", func(t *testing.T) {
		testData := []byte("Test hash data")
		
		hash1 := Hash(testData)
		hash2 := Hash(testData)
		
		if hash1 != hash2 {
			t.Error("Hash function should be deterministic")
		}
		
		// Test different data produces different hash
		differentData := []byte("Different test data")
		hash3 := Hash(differentData)
		
		if hash1 == hash3 {
			t.Error("Different data should produce different hashes")
		}
	})
}

// TestCryptographicPerformance 测试加密操作性能
func TestCryptographicPerformance(t *testing.T) {
	const iterations = 100
	
	t.Run("KeyGenerationPerformance", func(t *testing.T) {
		start := time.Now()
		
		for i := 0; i < iterations; i++ {
			_, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("Key generation failed at iteration %d: %v", i, err)
			}
		}
		
		duration := time.Since(start)
		avgTime := duration / iterations
		
		t.Logf("Generated %d keypairs in %v (avg: %v per keypair)", iterations, duration, avgTime)
		
		// 性能要求：每个密钥对生成应该在50μs内完成
		if avgTime > 50*time.Microsecond {
			t.Errorf("Key generation too slow: %v > 50μs", avgTime)
		}
	})
	
	t.Run("SignaturePerformance", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}
		
		testData := []byte("Performance test data")
		
		start := time.Now()
		
		for i := 0; i < iterations; i++ {
			signature := keyPair.Sign(testData)
			if signature.IsEmpty() {
				t.Fatalf("Signature empty at iteration %d", i)
			}
		}
		
		duration := time.Since(start)
		avgTime := duration / iterations
		
		t.Logf("Generated %d signatures in %v (avg: %v per signature)", iterations, duration, avgTime)
		
		// 性能要求：每次签名应该在100μs内完成
		if avgTime > 100*time.Microsecond {
			t.Errorf("Signature generation too slow: %v > 100μs", avgTime)
		}
	})
	
	t.Run("VerificationPerformance", func(t *testing.T) {
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}
		
		testData := []byte("Performance test data")
		signature := keyPair.Sign(testData)
		
		start := time.Now()
		
		for i := 0; i < iterations; i++ {
			if !VerifySignature(keyPair.PublicKey, testData, signature) {
				t.Fatalf("Signature verification failed at iteration %d", i)
			}
		}
		
		duration := time.Since(start)
		avgTime := duration / iterations
		
		t.Logf("Verified %d signatures in %v (avg: %v per verification)", iterations, duration, avgTime)
		
		// 性能要求：每次验证应该在100μs内完成
		if avgTime > 100*time.Microsecond {
			t.Errorf("Signature verification too slow: %v > 100μs", avgTime)
		}
	})
	
	t.Run("HashPerformance", func(t *testing.T) {
		testData := []byte("Hash performance test data")
		
		start := time.Now()
		
		for i := 0; i < iterations*10; i++ { // 更多次数测试哈希性能
			hash := Hash(testData)
			if hash.IsEmpty() {
				t.Fatalf("Hash empty at iteration %d", i)
			}
		}
		
		duration := time.Since(start)
		avgTime := duration / (iterations * 10)
		
		t.Logf("Generated %d hashes in %v (avg: %v per hash)", iterations*10, duration, avgTime)
		
		// 性能要求：每次哈希应该在1μs内完成
		if avgTime > time.Microsecond {
			t.Errorf("Hash calculation too slow: %v > 1μs", avgTime)
		}
	})
}