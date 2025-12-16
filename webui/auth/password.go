// Package auth provides authentication molecules for the web UI.
// This file contains the password hasher molecule for secure password management.
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Password hashing configuration constants
const (
	// DefaultCost is the bcrypt cost factor for password hashing.
	// Cost 12 provides a good balance between security and performance.
	// At cost 12, hashing takes ~250ms on modern hardware.
	// Increase this value as hardware improves.
	DefaultCost = 12

	// MinCost is the minimum acceptable bcrypt cost factor.
	// Lower costs are insecure and should not be used in production.
	MinCost = 10

	// MaxCost is the maximum bcrypt cost factor.
	// Higher costs are supported by bcrypt but may be impractically slow.
	MaxCost = 31
)

// Error definitions for password operations
var (
	// ErrEmptyPassword is returned when attempting to hash an empty password.
	ErrEmptyPassword = errors.New("password cannot be empty")

	// ErrPasswordMismatch is returned when password verification fails.
	// This error intentionally does not reveal whether the hash was valid.
	ErrPasswordMismatch = errors.New("password does not match")

	// ErrInvalidHash is returned when the hash format is invalid.
	ErrInvalidHash = errors.New("invalid password hash format")

	// ErrCostTooLow is returned when the hash cost is below MinCost.
	ErrCostTooLow = errors.New("hash cost is below minimum acceptable value")
)

// HashPassword creates a bcrypt hash of the given password.
// The hash includes a random salt and the cost factor, making it safe
// for direct storage in a database.
//
// This molecule composes:
//   - bcrypt.GenerateFromPassword for secure hashing
//   - DefaultCost for appropriate security/performance balance
//
// Security properties:
//   - Uses bcrypt's built-in salt generation (crypto/rand)
//   - Cost factor is embedded in the hash for future verification
//   - Hash output is 60 bytes in standard bcrypt format
//
// Parameters:
//   - password: The plaintext password to hash (must not be empty)
//
// Returns:
//   - string: The bcrypt hash (safe for storage)
//   - error: ErrEmptyPassword if password is empty, or bcrypt error
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// HashPasswordWithCost creates a bcrypt hash with a specific cost factor.
// Use this when you need to control the cost factor explicitly.
//
// Parameters:
//   - password: The plaintext password to hash
//   - cost: The bcrypt cost factor (MinCost to MaxCost)
//
// Returns:
//   - string: The bcrypt hash
//   - error: ErrEmptyPassword, or error if cost is invalid
func HashPasswordWithCost(password string, cost int) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	// bcrypt validates cost internally, but we add bounds checking
	if cost < MinCost || cost > MaxCost {
		return "", bcrypt.InvalidCostError(cost)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// VerifyPassword compares a plaintext password with a bcrypt hash.
// This function uses constant-time comparison to prevent timing attacks.
//
// This molecule composes:
//   - bcrypt.CompareHashAndPassword for timing-safe comparison
//   - Error normalization for consistent error handling
//
// Security properties:
//   - Constant-time comparison prevents timing attacks
//   - Does not reveal whether hash format was valid on mismatch
//
// Parameters:
//   - password: The plaintext password to verify
//   - hash: The bcrypt hash to compare against
//
// Returns:
//   - error: nil if password matches, ErrPasswordMismatch if not
func VerifyPassword(password, hash string) error {
	if password == "" {
		return ErrEmptyPassword
	}

	if hash == "" {
		return ErrInvalidHash
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		// Don't expose internal bcrypt errors - they could leak info
		return ErrPasswordMismatch
	}

	return nil
}

// NeedsRehash checks if a password hash should be upgraded to a higher cost.
// This should be called after successful password verification to detect
// hashes that were created with an older (lower) cost factor.
//
// Use this for gradual security upgrades: when NeedsRehash returns true,
// re-hash the password with HashPassword and store the new hash.
//
// Parameters:
//   - hash: The bcrypt hash to check
//   - targetCost: The minimum acceptable cost factor
//
// Returns:
//   - bool: true if the hash cost is below targetCost
func NeedsRehash(hash string, targetCost int) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		// Invalid hash format - should be rehashed
		return true
	}

	return cost < targetCost
}

// NeedsRehashDefault checks if a hash needs upgrading to DefaultCost.
// This is a convenience function that uses DefaultCost as the target.
//
// Parameters:
//   - hash: The bcrypt hash to check
//
// Returns:
//   - bool: true if the hash cost is below DefaultCost
func NeedsRehashDefault(hash string) bool {
	return NeedsRehash(hash, DefaultCost)
}

// GetHashCost extracts the cost factor from a bcrypt hash.
// This is useful for auditing and monitoring hash security.
//
// Parameters:
//   - hash: The bcrypt hash to inspect
//
// Returns:
//   - int: The cost factor used to create the hash
//   - error: ErrInvalidHash if the hash format is invalid
func GetHashCost(hash string) (int, error) {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return 0, ErrInvalidHash
	}
	return cost, nil
}

// ValidateHashStrength checks if a hash meets minimum security requirements.
// This can be used to audit existing password hashes.
//
// Parameters:
//   - hash: The bcrypt hash to validate
//
// Returns:
//   - error: nil if hash meets requirements, or an appropriate error
func ValidateHashStrength(hash string) error {
	if hash == "" {
		return ErrInvalidHash
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return ErrInvalidHash
	}

	if cost < MinCost {
		return ErrCostTooLow
	}

	return nil
}

// IsValidHash checks if a string is a valid bcrypt hash format.
// This does not verify the password, only that the hash is well-formed.
//
// Parameters:
//   - hash: The string to check
//
// Returns:
//   - bool: true if the string is a valid bcrypt hash format
func IsValidHash(hash string) bool {
	_, err := bcrypt.Cost([]byte(hash))
	return err == nil
}
