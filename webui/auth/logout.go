// Package auth provides authentication components for the web UI.
// This file contains the logout handler that clears sessions and cookies.
package auth

import (
	"net/http"

	"go.uber.org/zap"
)

// LogoutHandler creates an HTTP handler that handles user logout.
// The handler:
//  1. Extracts the session ID from the cookie
//  2. Destroys the session in the store
//  3. Clears the session cookie
//  4. Redirects to the login page
//
// This handler accepts both GET and POST methods for flexibility:
//   - GET: For simple logout links
//   - POST: For logout forms with CSRF protection (if implemented)
//
// The handler is idempotent - calling it multiple times or without a valid
// session will still redirect to login without error.
//
// Parameters:
//   - m: The AuthMiddleware containing session store and logger
//
// Returns:
//   - http.HandlerFunc: Handler for the /logout endpoint
//
// Usage:
//
//	mux.HandleFunc("/logout", auth.LogoutHandler(authMiddleware))
func LogoutHandler(m *AuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET and POST methods
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			m.logger.Debug("logout: invalid method",
				zap.String("method", r.Method),
				zap.String("ip", getClientIP(r)),
			)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Try to extract session ID from cookie
		sessionID, err := ParseSessionCookieDefault(r)
		if err != nil {
			// No valid session cookie - still redirect to login
			// This is not an error condition; user may already be logged out
			m.logger.Debug("logout: no session cookie found",
				zap.String("ip", getClientIP(r)),
			)
		} else {
			// Destroy the session
			// DestroySession handles non-existent sessions gracefully
			m.DestroySession(sessionID)
			m.logger.Info("logout: session destroyed",
				zap.String("session_id", truncateSessionID(sessionID)),
				zap.String("ip", getClientIP(r)),
			)
		}

		// Clear the session cookie by setting MaxAge=-1
		clearCookie := ClearSessionCookieDefault()
		http.SetCookie(w, clearCookie)

		// Redirect to login page
		// Use 302 (Found) for GET, 303 (See Other) for POST
		// to ensure browser doesn't resubmit the form
		redirectCode := http.StatusFound
		if r.Method == http.MethodPost {
			redirectCode = http.StatusSeeOther
		}

		http.Redirect(w, r, "/login", redirectCode)
	}
}

// truncateSessionID returns a truncated session ID for safe logging.
// Shows only the first 8 characters followed by "..." for privacy.
func truncateSessionID(sessionID string) string {
	if len(sessionID) <= 8 {
		return sessionID + "..."
	}
	return sessionID[:8] + "..."
}
