package types

const (
	// ModuleName defines the module name
	ModuleName = "shardmatrix"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_shardmatrix"
)

var (
	ParamsKey = []byte("p_shardmatrix")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
