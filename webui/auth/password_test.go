package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password",
			password: "securePassword123!",
			wantErr:  nil,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  ErrEmptyPassword,
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 72), // bcrypt max is 72 bytes
			wantErr:  nil,
		},
		{
			name:     "password with special characters",
			password: "p@$$w0rd!#$%^&*()",
			wantErr:  nil,
		},
		{
			name:     "unicode password",
			password: "passwordwith symbols",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify hash is valid bcrypt format
			if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
				t.Errorf("hash should be bcrypt format, got: %s", hash[:10])
			}

			// Verify hash can be used to verify the password
			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
			if err != nil {
				t.Errorf("hash should verify against original password: %v", err)
			}
		})
	}
}

func TestHashPasswordWithCost(t *testing.T) {
	password := "testPassword123"

	tests := []struct {
		name    string
		cost    int
		wantErr bool
	}{
		{"minimum cost", MinCost, false},
		{"default cost", DefaultCost, false},
		{"high cost", 14, false},
		{"too low cost", MinCost - 1, true},
		{"too high cost", MaxCost + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPasswordWithCost(password, tt.cost)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error for invalid cost")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the cost is correct in the hash
			cost, err := bcrypt.Cost([]byte(hash))
			if err != nil {
				t.Fatalf("failed to get cost from hash: %v", err)
			}
			if cost != tt.cost {
				t.Errorf("expected cost %d, got %d", tt.cost, cost)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	// Create a known hash for testing
	password := "correctPassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create test hash: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  error
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			wantErr:  nil,
		},
		{
			name:     "wrong password",
			password: "wrongPassword",
			hash:     hash,
			wantErr:  ErrPasswordMismatch,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			wantErr:  ErrEmptyPassword,
		},
		{
			name:     "empty hash",
			password: password,
			hash:     "",
			wantErr:  ErrInvalidHash,
		},
		{
			name:     "invalid hash format",
			password: password,
			hash:     "not-a-valid-bcrypt-hash",
			wantErr:  ErrPasswordMismatch, // Should not reveal hash format issues
		},
		{
			name:     "case sensitive password",
			password: "CORRECTPASSWORD123",
			hash:     hash,
			wantErr:  ErrPasswordMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.password, tt.hash)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNeedsRehash(t *testing.T) {
	// Create hashes with different costs
	password := "testPassword"

	lowCostHash, err := HashPasswordWithCost(password, MinCost)
	if err != nil {
		t.Fatalf("failed to create low cost hash: %v", err)
	}

	defaultCostHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create default cost hash: %v", err)
	}

	tests := []struct {
		name       string
		hash       string
		targetCost int
		needsHash  bool
	}{
		{
			name:       "low cost hash needs upgrade",
			hash:       lowCostHash,
			targetCost: DefaultCost,
			needsHash:  true,
		},
		{
			name:       "default cost hash is current",
			hash:       defaultCostHash,
			targetCost: DefaultCost,
			needsHash:  false,
		},
		{
			name:       "hash at target cost",
			hash:       lowCostHash,
			targetCost: MinCost,
			needsHash:  false,
		},
		{
			name:       "invalid hash returns true",
			hash:       "invalid",
			targetCost: DefaultCost,
			needsHash:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsRehash(tt.hash, tt.targetCost)
			if result != tt.needsHash {
				t.Errorf("expected NeedsRehash=%v, got %v", tt.needsHash, result)
			}
		})
	}
}

func TestNeedsRehashDefault(t *testing.T) {
	// Create a low-cost hash that should need rehashing
	lowCostHash, _ := HashPasswordWithCost("password", MinCost)

	if !NeedsRehashDefault(lowCostHash) {
		t.Error("low cost hash should need rehashing to default cost")
	}

	// Create a default cost hash that should not need rehashing
	defaultHash, _ := HashPassword("password")

	if NeedsRehashDefault(defaultHash) {
		t.Error("default cost hash should not need rehashing")
	}
}

func TestGetHashCost(t *testing.T) {
	tests := []struct {
		name         string
		createHash   func() string
		expectedCost int
		wantErr      error
	}{
		{
			name: "default cost",
			createHash: func() string {
				h, _ := HashPassword("test")
				return h
			},
			expectedCost: DefaultCost,
			wantErr:      nil,
		},
		{
			name: "minimum cost",
			createHash: func() string {
				h, _ := HashPasswordWithCost("test", MinCost)
				return h
			},
			expectedCost: MinCost,
			wantErr:      nil,
		},
		{
			name: "invalid hash",
			createHash: func() string {
				return "not-a-hash"
			},
			expectedCost: 0,
			wantErr:      ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := tt.createHash()
			cost, err := GetHashCost(hash)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cost != tt.expectedCost {
				t.Errorf("expected cost %d, got %d", tt.expectedCost, cost)
			}
		})
	}
}

func TestValidateHashStrength(t *testing.T) {
	tests := []struct {
		name    string
		hash    string
		wantErr error
	}{
		{
			name: "valid default cost hash",
			hash: func() string {
				h, _ := HashPassword("test")
				return h
			}(),
			wantErr: nil,
		},
		{
			name: "valid minimum cost hash",
			hash: func() string {
				h, _ := HashPasswordWithCost("test", MinCost)
				return h
			}(),
			wantErr: nil,
		},
		{
			name:    "empty hash",
			hash:    "",
			wantErr: ErrInvalidHash,
		},
		{
			name:    "invalid hash format",
			hash:    "not-a-valid-hash",
			wantErr: ErrInvalidHash,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHashStrength(tt.hash)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsValidHash(t *testing.T) {
	validHash, _ := HashPassword("test")

	tests := []struct {
		name  string
		hash  string
		valid bool
	}{
		{"valid bcrypt hash", validHash, true},
		{"empty string", "", false},
		{"random string", "not-a-hash", false},
		{"partial hash", "$2a$12$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidHash(tt.hash)
			if result != tt.valid {
				t.Errorf("IsValidHash(%q) = %v, want %v", tt.hash, result, tt.valid)
			}
		})
	}
}

func TestHashUniqueness(t *testing.T) {
	// Same password should produce different hashes (due to random salt)
	password := "samePassword123"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create hash1: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to create hash2: %v", err)
	}

	if hash1 == hash2 {
		t.Error("same password should produce different hashes due to random salt")
	}

	// Both hashes should still verify against the original password
	if err := VerifyPassword(password, hash1); err != nil {
		t.Errorf("hash1 should verify: %v", err)
	}
	if err := VerifyPassword(password, hash2); err != nil {
		t.Errorf("hash2 should verify: %v", err)
	}
}

// BenchmarkHashPassword measures hashing performance
func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

// BenchmarkVerifyPassword measures verification performance
func BenchmarkVerifyPassword(b *testing.B) {
	password := "benchmarkPassword123!"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(password, hash)
	}
}
