// Package auth provides authentication components for the web UI.
// This file contains the login handler that handles both GET (render form)
// and POST (authenticate) requests for the /login endpoint.
package auth

import (
	"net/http"
	"time"

	"go_backend/webui"

	"go.uber.org/zap"
)

// Default login handler configuration
const (
	// FailedLoginDelay is the delay added after a failed login attempt
	// to slow down brute force attacks and prevent timing attacks.
	FailedLoginDelay = 1 * time.Second

	// SuccessRedirect is the default path to redirect after successful login.
	SuccessRedirect = "/"

	// LoginPath is the path for the login page.
	LoginPath = "/login"
)

// LoginHandler creates an HTTP handler that handles user login.
// The handler supports both GET and POST methods:
//
// GET /login:
//   - Renders the login form
//   - Displays error message from query parameter if present
//
// POST /login:
//  1. Checks rate limit for the client IP
//  2. Extracts password from form data
//  3. Verifies the password
//  4. On success: creates session, sets cookie, redirects to dashboard
//  5. On failure: adds 1-second delay, records attempt, redirects with error
//
// Rate limiting prevents brute force attacks by blocking IPs that make
// too many failed attempts within a time window.
//
// Parameters:
//   - m: The AuthMiddleware containing password hash, session store, rate limiter, and logger
//
// Returns:
//   - http.HandlerFunc: Handler for the /login endpoint
//
// Usage:
//
//	mux.HandleFunc("/login", auth.LoginHandler(authMiddleware))
func LoginHandler(m *AuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleLoginGET(w, r, m)
		case http.MethodPost:
			handleLoginPOST(w, r, m)
		default:
			m.logger.Debug("login: invalid method",
				zap.String("method", r.Method),
				zap.String("ip", getClientIP(r)),
			)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleLoginGET renders the login form page.
func handleLoginGET(w http.ResponseWriter, r *http.Request, m *AuthMiddleware) {
	// Check if user already has a valid session
	sessionID, err := ParseSessionCookieDefault(r)
	if err == nil {
		// Session cookie exists, check if valid
		if _, err := m.GetSession(sessionID); err == nil {
			// Already logged in, redirect to dashboard
			m.logger.Debug("login GET: user already logged in, redirecting",
				zap.String("ip", getClientIP(r)),
			)
			http.Redirect(w, r, SuccessRedirect, http.StatusFound)
			return
		}
	}

	// Render the login page using the webui package
	webui.HandleLoginPage(w, r)
}

// handleLoginPOST handles the form submission for authentication.
func handleLoginPOST(w http.ResponseWriter, r *http.Request, m *AuthMiddleware) {
	clientIP := getClientIP(r)

	// Step 1: Check rate limit
	if !m.CheckRateLimit(w, clientIP) {
		// Response already sent by CheckRateLimit (429 Too Many Requests)
		return
	}

	// Step 2: Parse form data
	if err := r.ParseForm(); err != nil {
		m.logger.Debug("login POST: failed to parse form",
			zap.String("ip", clientIP),
			zap.Error(err),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")

	// Step 3: Validate password is not empty
	if password == "" {
		m.logger.Debug("login POST: empty password",
			zap.String("ip", clientIP),
		)
		// Add delay to prevent timing attacks
		time.Sleep(FailedLoginDelay)
		redirectWithError(w, r, "Password is required")
		return
	}

	// Step 4: Verify password
	if err := m.VerifyPassword(password); err != nil {
		// Failed login - record the attempt and add delay
		m.RecordFailedAttempt(clientIP)

		m.logger.Info("login POST: authentication failed",
			zap.String("ip", clientIP),
		)

		// Add delay to slow down brute force attacks
		time.Sleep(FailedLoginDelay)

		redirectWithError(w, r, "Invalid password")
		return
	}

	// Step 5: Success - create session
	_, cookie, err := m.CreateSession()
	if err != nil {
		m.logger.Error("login POST: failed to create session",
			zap.String("ip", clientIP),
			zap.Error(err),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Step 6: Reset rate limit for this IP (successful login)
	m.ResetRateLimit(clientIP)

	// Step 7: Set the session cookie
	http.SetCookie(w, cookie)

	m.logger.Info("login POST: authentication successful",
		zap.String("ip", clientIP),
	)

	// Step 8: Redirect to dashboard
	// Use 303 See Other to prevent form resubmission on refresh
	http.Redirect(w, r, SuccessRedirect, http.StatusSeeOther)
}

// redirectWithError redirects to the login page with an error message.
// The error is passed as a query parameter to be displayed by the login form.
func redirectWithError(w http.ResponseWriter, r *http.Request, errMsg string) {
	// Encode the error message in the query string
	http.Redirect(w, r, LoginPath+"?error="+errMsg, http.StatusSeeOther)
}

// LoginHandlerWithRedirect creates a login handler with a custom success redirect path.
// This is useful when you want to redirect to a different page after login.
//
// Parameters:
//   - m: The AuthMiddleware containing authentication components
//   - successPath: The path to redirect to after successful login
//
// Returns:
//   - http.HandlerFunc: Handler for the /login endpoint
//
// Usage:
//
//	mux.HandleFunc("/login", auth.LoginHandlerWithRedirect(authMiddleware, "/dashboard"))
func LoginHandlerWithRedirect(m *AuthMiddleware, successPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleLoginGETWithRedirect(w, r, m, successPath)
		case http.MethodPost:
			handleLoginPOSTWithRedirect(w, r, m, successPath)
		default:
			m.logger.Debug("login: invalid method",
				zap.String("method", r.Method),
				zap.String("ip", getClientIP(r)),
			)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleLoginGETWithRedirect renders the login form page with custom redirect.
func handleLoginGETWithRedirect(w http.ResponseWriter, r *http.Request, m *AuthMiddleware, successPath string) {
	// Check if user already has a valid session
	sessionID, err := ParseSessionCookieDefault(r)
	if err == nil {
		// Session cookie exists, check if valid
		if _, err := m.GetSession(sessionID); err == nil {
			// Already logged in, redirect to success path
			m.logger.Debug("login GET: user already logged in, redirecting",
				zap.String("ip", getClientIP(r)),
				zap.String("redirect", successPath),
			)
			http.Redirect(w, r, successPath, http.StatusFound)
			return
		}
	}

	// Render the login page using the webui package
	webui.HandleLoginPage(w, r)
}

// handleLoginPOSTWithRedirect handles the form submission with custom success redirect.
func handleLoginPOSTWithRedirect(w http.ResponseWriter, r *http.Request, m *AuthMiddleware, successPath string) {
	clientIP := getClientIP(r)

	// Step 1: Check rate limit
	if !m.CheckRateLimit(w, clientIP) {
		// Response already sent by CheckRateLimit (429 Too Many Requests)
		return
	}

	// Step 2: Parse form data
	if err := r.ParseForm(); err != nil {
		m.logger.Debug("login POST: failed to parse form",
			zap.String("ip", clientIP),
			zap.Error(err),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")

	// Step 3: Validate password is not empty
	if password == "" {
		m.logger.Debug("login POST: empty password",
			zap.String("ip", clientIP),
		)
		// Add delay to prevent timing attacks
		time.Sleep(FailedLoginDelay)
		redirectWithError(w, r, "Password is required")
		return
	}

	// Step 4: Verify password
	if err := m.VerifyPassword(password); err != nil {
		// Failed login - record the attempt and add delay
		m.RecordFailedAttempt(clientIP)

		m.logger.Info("login POST: authentication failed",
			zap.String("ip", clientIP),
		)

		// Add delay to slow down brute force attacks
		time.Sleep(FailedLoginDelay)

		redirectWithError(w, r, "Invalid password")
		return
	}

	// Step 5: Success - create session
	_, cookie, err := m.CreateSession()
	if err != nil {
		m.logger.Error("login POST: failed to create session",
			zap.String("ip", clientIP),
			zap.Error(err),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Step 6: Reset rate limit for this IP (successful login)
	m.ResetRateLimit(clientIP)

	// Step 7: Set the session cookie
	http.SetCookie(w, cookie)

	m.logger.Info("login POST: authentication successful",
		zap.String("ip", clientIP),
		zap.String("redirect", successPath),
	)

	// Step 8: Redirect to success path
	// Use 303 See Other to prevent form resubmission on refresh
	http.Redirect(w, r, successPath, http.StatusSeeOther)
}
