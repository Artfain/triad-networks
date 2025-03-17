package types

const (
	// ModuleName defines the module name
	ModuleName = "triad"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_triad"
)

var (
	ParamsKey = []byte("p_triad")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
