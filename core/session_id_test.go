package core

import (
	"encoding/base64"
	"testing"
)

func TestGenerateSessionID_ReturnsValidBase64(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() returned error: %v", err)
	}

	// Verify it's valid base64 URL encoding
	decoded, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(id)
	if err != nil {
		t.Errorf("Generated ID is not valid base64 URL encoding: %v", err)
	}

	// Verify decoded length is 32 bytes
	if len(decoded) != SessionIDLength {
		t.Errorf("Decoded ID length = %d, want %d", len(decoded), SessionIDLength)
	}
}

func TestGenerateSessionID_ReturnsUniqueIDs(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	const iterations = 100

	for i := 0; i < iterations; i++ {
		id, err := GenerateSessionID()
		if err != nil {
			t.Fatalf("GenerateSessionID() returned error on iteration %d: %v", i, err)
		}

		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateSessionID_HasExpectedLength(t *testing.T) {
	id, err := GenerateSessionID()
	if err != nil {
		t.Fatalf("GenerateSessionID() returned error: %v", err)
	}

	// 32 bytes base64 encoded without padding = 43 characters
	expectedLen := 43
	if len(id) != expectedLen {
		t.Errorf("Generated ID length = %d, want %d", len(id), expectedLen)
	}
}
