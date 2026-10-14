package domain

// Limits bound the v1 JSON envelope. They are shape checks; the relay never
// interprets plaintext.
type Limits struct {
	MaxBatch      int
	MaxCiphertext int
	MaxAAD        int
	MinNonce      int
	MaxNonce      int
	MinTag        int
	MaxTag        int
	MaxParents    int
	SchemaVersion int
	MaxAlgorithm  int
}

// DefaultLimits is the v1 protocol fixture bound.
func DefaultLimits() Limits {
	return Limits{
		MaxBatch:      32,
		MaxCiphertext: 256 * 1024,
		MaxAAD:        4096,
		MinNonce:      12,
		MaxNonce:      64,
		MinTag:        16,
		MaxTag:        64,
		MaxParents:    16,
		SchemaVersion: 1,
		MaxAlgorithm:  64,
	}
}
