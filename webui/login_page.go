// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the login page rendering functionality.
package webui

import (
	"html/template"
	"io"
	"net/http"
)

// loginPageHTML is the embedded HTML template for the login page.
// It includes all CSS inline for simplicity and single-file deployment.
const loginPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CanvusLocalLLM - Login</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
            color: #ffffff;
        }

        .login-container {
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 16px;
            padding: 48px;
            width: 100%;
            max-width: 400px;
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
        }

        .login-header {
            text-align: center;
            margin-bottom: 32px;
        }

        .login-header h1 {
            font-size: 28px;
            font-weight: 600;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #60a5fa 0%, #a78bfa 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .login-header p {
            font-size: 14px;
            color: rgba(255, 255, 255, 0.6);
        }

        .login-form {
            display: flex;
            flex-direction: column;
            gap: 24px;
        }

        .form-group {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .form-group label {
            font-size: 14px;
            font-weight: 500;
            color: rgba(255, 255, 255, 0.8);
        }

        .form-group input {
            padding: 14px 16px;
            font-size: 16px;
            border: 1px solid rgba(255, 255, 255, 0.15);
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.08);
            color: #ffffff;
            outline: none;
            transition: border-color 0.2s ease, background-color 0.2s ease;
        }

        .form-group input:focus {
            border-color: #60a5fa;
            background: rgba(255, 255, 255, 0.12);
        }

        .form-group input::placeholder {
            color: rgba(255, 255, 255, 0.4);
        }

        .submit-btn {
            padding: 14px 24px;
            font-size: 16px;
            font-weight: 600;
            color: #ffffff;
            background: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%);
            border: none;
            border-radius: 8px;
            cursor: pointer;
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .submit-btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 25px -5px rgba(59, 130, 246, 0.5);
        }

        .submit-btn:active {
            transform: translateY(0);
        }

        .submit-btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }

        .error-message {
            padding: 12px 16px;
            font-size: 14px;
            color: #fca5a5;
            background: rgba(239, 68, 68, 0.15);
            border: 1px solid rgba(239, 68, 68, 0.3);
            border-radius: 8px;
            text-align: center;
            display: {{if .Error}}block{{else}}none{{end}};
        }

        .footer {
            text-align: center;
            margin-top: 24px;
            font-size: 12px;
            color: rgba(255, 255, 255, 0.4);
        }

        @media (max-width: 480px) {
            .login-container {
                margin: 16px;
                padding: 32px 24px;
            }

            .login-header h1 {
                font-size: 24px;
            }
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1>CanvusLocalLLM</h1>
            <p>Enter your password to access the dashboard</p>
        </div>

        <form class="login-form" method="POST" action="/login">
            <div class="error-message">{{.Error}}</div>

            <div class="form-group">
                <label for="password">Password</label>
                <input
                    type="password"
                    id="password"
                    name="password"
                    placeholder="Enter your password"
                    required
                    autofocus
                >
            </div>

            <button type="submit" class="submit-btn">Sign In</button>
        </form>

        <div class="footer">
            <p>Secured with WEBUI_PWD from configuration</p>
        </div>
    </div>
</body>
</html>`

// LoginPageData holds the data passed to the login page template.
type LoginPageData struct {
	// Error contains an error message to display, empty if no error
	Error string
}

// loginTemplate is the parsed login page template.
// It's parsed once and reused for all requests.
var loginTemplate = template.Must(template.New("login").Parse(loginPageHTML))

// RenderLoginPage writes the login page HTML to the provided writer.
// This is used to serve the login page for unauthenticated users.
//
// Parameters:
//   - w: The writer to output the rendered HTML (typically http.ResponseWriter)
//   - data: The template data containing any error message to display
//
// Returns an error if template execution fails.
func RenderLoginPage(w io.Writer, data LoginPageData) error {
	return loginTemplate.Execute(w, data)
}

// HandleLoginPage is an HTTP handler that renders the login page.
// It serves GET requests for /login and displays any error messages.
func HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	data := LoginPageData{}

	// Check for error query parameter (set by failed login attempt)
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		data.Error = errMsg
	}

	if err := RenderLoginPage(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
