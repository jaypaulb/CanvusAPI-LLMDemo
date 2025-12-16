package sdruntime

import (
	"testing"
)

func TestRandomSeed_NonNegative(t *testing.T) {
	// Generate multiple seeds and verify they're all non-negative
	for i := 0; i < 100; i++ {
		seed := RandomSeed()
		if seed < 0 {
			t.Errorf("seed should be non-negative, got: %d", seed)
		}
	}
}

func TestRandomSeed_Randomness(t *testing.T) {
	// Generate seeds and verify they're not all the same
	seeds := make(map[int64]bool)
	for i := 0; i < 10; i++ {
		seed := RandomSeed()
		seeds[seed] = true
	}

	// With 10 random int64 values, we should have multiple unique values
	// (probability of collision is astronomically low)
	if len(seeds) < 5 {
		t.Errorf("expected multiple unique seeds, got only %d unique values", len(seeds))
	}
}

func TestRandomSeed_ValidRange(t *testing.T) {
	// Verify seed is within valid int64 range (non-negative)
	seed := RandomSeed()
	if seed < 0 {
		t.Errorf("seed must be >= 0, got: %d", seed)
	}
}
