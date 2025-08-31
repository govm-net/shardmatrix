package validator

import "github.com/govm-net/shardmatrix/pkg/types"

// Helper functions for testing

// testAddress creates an address from string for tests
func testAddress(s string) types.Address {
	// Use AddressFromPublicKey for testing
	return types.AddressFromPublicKey([]byte(s))
}
