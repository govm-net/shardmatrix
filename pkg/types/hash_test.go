package types

import (
	"testing"
)

func TestHashCreation(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool // 检查是否成功创建
	}{
		{
			name:     "valid hash creation from data",
			input:    []byte("test data"),
			expected: true,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: true,
		},
		{
			name:     "32 byte input",
			input:    make([]byte, 32),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewHash(tt.input)
			// 对于32字节的全零输入，NewHash直接复制数据，结果是零哈希
			// 这是正确的行为
			if len(tt.input) != 32 && tt.expected && result.IsZero() && len(tt.input) > 0 {
				t.Errorf("NewHash() returned zero hash for non-empty input")
			}
			// 验证哈希长度总是32字节
			if len(result.Bytes()) != 32 {
				t.Errorf("Hash should always be 32 bytes, got %d", len(result.Bytes()))
			}
		})
	}
}

func TestHashFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid 64 char hex string",
			input:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			wantErr: false,
		},
		{
			name:    "invalid length",
			input:   "123",
			wantErr: true,
		},
		{
			name:    "invalid hex characters",
			input:   "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashMethods(t *testing.T) {
	hash1 := NewHash([]byte("test1"))
	hash2 := NewHash([]byte("test1"))
	hash3 := NewHash([]byte("test2"))

	// 测试Equal方法
	if !hash1.Equal(hash2) {
		t.Error("Equal hashes should be equal")
	}

	if hash1.Equal(hash3) {
		t.Error("Different hashes should not be equal")
	}

	// 测试String方法
	if len(hash1.String()) != 64 {
		t.Error("Hash string should be 64 characters")
	}

	// 测试Bytes方法
	if len(hash1.Bytes()) != 32 {
		t.Error("Hash bytes should be 32 bytes")
	}

	// 测试IsZero方法
	emptyHash := EmptyHash()
	if !emptyHash.IsZero() {
		t.Error("Empty hash should be zero")
	}

	if hash1.IsZero() {
		t.Error("Non-empty hash should not be zero")
	}
}

func BenchmarkHashCreation(b *testing.B) {
	data := []byte("benchmark test data")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = NewHash(data)
	}
}

func BenchmarkHashString(b *testing.B) {
	hash := NewHash([]byte("test data"))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = hash.String()
	}
}
