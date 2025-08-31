package types

import (
	"testing"
)

func TestAddressCreation(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{
			name:     "valid 20 byte input",
			input:    make([]byte, 20),
			expected: true,
		},
		{
			name:     "longer input (truncated)",
			input:    make([]byte, 30),
			expected: true,
		},
		{
			name:     "shorter input (padded)",
			input:    make([]byte, 10),
			expected: true,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAddress(tt.input)
			if len(result.Bytes()) != 20 {
				t.Errorf("Address should always be 20 bytes, got %d", len(result.Bytes()))
			}
		})
	}
}

func TestAddressFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid address with 0x prefix",
			input:   "0x0123456789abcdef0123456789abcdef01234567",
			wantErr: false,
		},
		{
			name:    "valid address without 0x prefix",
			input:   "0123456789abcdef0123456789abcdef01234567",
			wantErr: false,
		},
		{
			name:    "invalid length",
			input:   "0x123",
			wantErr: true,
		},
		{
			name:    "invalid hex characters",
			input:   "0xzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AddressFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddressFromString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAddressFromPublicKey(t *testing.T) {
	pubKey := []byte("test public key")
	addr := AddressFromPublicKey(pubKey)

	if addr.IsZero() {
		t.Error("Address from public key should not be zero")
	}

	if len(addr.Bytes()) != 20 {
		t.Error("Address should be 20 bytes")
	}

	// 相同的公钥应该生成相同的地址
	addr2 := AddressFromPublicKey(pubKey)
	if !addr.Equal(addr2) {
		t.Error("Same public key should generate same address")
	}
}

func TestAddressMethods(t *testing.T) {
	addr1 := NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	addr2 := NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	addr3 := NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 21})

	// 测试Equal方法
	if !addr1.Equal(addr2) {
		t.Error("Equal addresses should be equal")
	}

	if addr1.Equal(addr3) {
		t.Error("Different addresses should not be equal")
	}

	// 测试String方法
	str := addr1.String()
	if len(str) != 42 { // 0x + 40 hex chars
		t.Errorf("Address string should be 42 characters, got %d", len(str))
	}

	if str[:2] != "0x" {
		t.Error("Address string should start with 0x")
	}

	// 测试Bytes方法
	if len(addr1.Bytes()) != 20 {
		t.Error("Address bytes should be 20 bytes")
	}

	// 测试IsZero方法
	emptyAddr := EmptyAddress()
	if !emptyAddr.IsZero() {
		t.Error("Empty address should be zero")
	}

	if addr1.IsZero() {
		t.Error("Non-empty address should not be zero")
	}
}

func BenchmarkAddressFromPublicKey(b *testing.B) {
	pubKey := []byte("test public key for benchmarking")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = AddressFromPublicKey(pubKey)
	}
}

func BenchmarkAddressString(b *testing.B) {
	addr := NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = addr.String()
	}
}
