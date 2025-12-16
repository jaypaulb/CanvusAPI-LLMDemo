package core

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// SessionIDLength is the number of random bytes used to generate session IDs.
// 32 bytes provides 256 bits of entropy, sufficient for cryptographic security.
const SessionIDLength = 32

// GenerateSessionID generates a cryptographically secure random session ID.
// Returns a base64 URL-encoded string of 32 random bytes (43 characters).
// This is a pure function that uses crypto/rand for cryptographic security.
//
// The returned string is safe for use in URLs and cookies without encoding.
func GenerateSessionID() (string, error) {
	bytes := make([]byte, SessionIDLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}
