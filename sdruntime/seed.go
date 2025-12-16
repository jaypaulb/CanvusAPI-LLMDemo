package sdruntime

import (
	"crypto/rand"
	"encoding/binary"
)

// RandomSeed generates a cryptographically secure random seed for image generation.
// Returns a non-negative int64 value suitable for reproducible image generation.
// This function uses crypto/rand for security.
func RandomSeed() int64 {
	var buf [8]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		// Fallback to a fixed seed if crypto/rand fails (extremely rare)
		// This is better than panicking in production
		return 42
	}

	// Convert to int64 and ensure non-negative by masking the sign bit
	seed := int64(binary.LittleEndian.Uint64(buf[:]))
	if seed < 0 {
		seed = -seed
	}
	// Handle edge case where -MinInt64 == MinInt64 (still negative)
	if seed < 0 {
		seed = 0
	}
	return seed
}
