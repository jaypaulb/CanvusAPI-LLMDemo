package handlers

import (
	"net/http"
	"testing"
	"time"
)

func TestNewAIClientFactory(t *testing.T) {
	factory := NewAIClientFactory()
	if factory == nil {
		t.Fatal("NewAIClientFactory() returned nil")
	}
}

func TestAIClientFactory_CreateClient(t *testing.T) {
	tests := []struct {
		name   string
		config AIClientConfig
	}{
		{
			name: "basic configuration",
			config: AIClientConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://api.example.com/v1",
			},
		},
		{
			name: "with fallback URL",
			config: AIClientConfig{
				APIKey:      "test-api-key",
				BaseURL:     "",
				FallbackURL: "https://fallback.example.com/v1",
			},
		},
		{
			name: "primary URL takes precedence",
			config: AIClientConfig{
				APIKey:      "test-api-key",
				BaseURL:     "https://primary.example.com/v1",
				FallbackURL: "https://fallback.example.com/v1",
			},
		},
		{
			name: "with custom HTTP client",
			config: AIClientConfig{
				APIKey:     "test-api-key",
				BaseURL:    "https://api.example.com/v1",
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			},
		},
		{
			name: "empty configuration",
			config: AIClientConfig{
				APIKey: "test-api-key",
			},
		},
	}

	factory := NewAIClientFactory()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := factory.CreateClient(tt.config)
			if client == nil {
				t.Error("CreateClient() returned nil")
			}
		})
	}
}

func TestAIClientFactory_CreateTextClient(t *testing.T) {
	factory := NewAIClientFactory()
	httpClient := &http.Client{Timeout: 30 * time.Second}

	tests := []struct {
		name       string
		apiKey     string
		textLLMURL string
		baseLLMURL string
	}{
		{
			name:       "primary URL only",
			apiKey:     "test-key",
			textLLMURL: "https://text.example.com/v1",
			baseLLMURL: "",
		},
		{
			name:       "fallback URL only",
			apiKey:     "test-key",
			textLLMURL: "",
			baseLLMURL: "https://base.example.com/v1",
		},
		{
			name:       "both URLs (primary wins)",
			apiKey:     "test-key",
			textLLMURL: "https://text.example.com/v1",
			baseLLMURL: "https://base.example.com/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := factory.CreateTextClient(tt.apiKey, tt.textLLMURL, tt.baseLLMURL, httpClient)
			if client == nil {
				t.Error("CreateTextClient() returned nil")
			}
		})
	}
}

func TestAIClientFactory_CreateImageClient(t *testing.T) {
	factory := NewAIClientFactory()
	httpClient := &http.Client{Timeout: 60 * time.Second}

	client := factory.CreateImageClient("test-key", "https://api.openai.com/v1", httpClient)
	if client == nil {
		t.Error("CreateImageClient() returned nil")
	}
}

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		primary  string
		fallback string
		want     string
	}{
		{
			name:     "primary non-empty",
			primary:  "https://primary.com",
			fallback: "https://fallback.com",
			want:     "https://primary.com",
		},
		{
			name:     "primary empty uses fallback",
			primary:  "",
			fallback: "https://fallback.com",
			want:     "https://fallback.com",
		},
		{
			name:     "both empty",
			primary:  "",
			fallback: "",
			want:     "",
		},
		{
			name:     "fallback empty",
			primary:  "https://primary.com",
			fallback: "",
			want:     "https://primary.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveBaseURL(tt.primary, tt.fallback)
			if got != tt.want {
				t.Errorf("ResolveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsLocalEndpoint(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "127.0.0.1",
			url:  "http://127.0.0.1:8080/v1",
			want: true,
		},
		{
			name: "localhost",
			url:  "http://localhost:8080/v1",
			want: true,
		},
		{
			name: "LOCALHOST uppercase",
			url:  "http://LOCALHOST:8080/v1",
			want: true,
		},
		{
			name: "0.0.0.0",
			url:  "http://0.0.0.0:8080/v1",
			want: true,
		},
		{
			name: "IPv6 localhost",
			url:  "http://[::1]:8080/v1",
			want: true,
		},
		{
			name: "external URL",
			url:  "https://api.openai.com/v1",
			want: false,
		},
		{
			name: "external IP",
			url:  "http://192.168.1.100:8080/v1",
			want: false,
		},
		{
			name: "localhost in path (not host)",
			url:  "https://api.example.com/localhost/v1",
			want: true, // Note: this is a quirk of simple substring matching
		},
		{
			name: "empty URL",
			url:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLocalEndpoint(tt.url)
			if got != tt.want {
				t.Errorf("IsLocalEndpoint(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		str    string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			str:    "localhost",
			substr: "localhost",
			want:   true,
		},
		{
			name:   "case insensitive",
			str:    "LOCALHOST",
			substr: "localhost",
			want:   true,
		},
		{
			name:   "mixed case",
			str:    "LocalHost",
			substr: "LOCALHOST",
			want:   true,
		},
		{
			name:   "substring at start",
			str:    "localhost:8080",
			substr: "local",
			want:   true,
		},
		{
			name:   "substring at end",
			str:    "http://localhost",
			substr: "host",
			want:   true,
		},
		{
			name:   "no match",
			str:    "example.com",
			substr: "localhost",
			want:   false,
		},
		{
			name:   "empty substring",
			str:    "localhost",
			substr: "",
			want:   true,
		},
		{
			name:   "empty string",
			str:    "",
			substr: "test",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsIgnoreCase(tt.str, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.str, tt.substr, got, tt.want)
			}
		})
	}
}

func TestToLowerASCII(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uppercase",
			input: "HELLO",
			want:  "hello",
		},
		{
			name:  "lowercase",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "mixed case",
			input: "HeLLo WoRLD",
			want:  "hello world",
		},
		{
			name:  "numbers unchanged",
			input: "Test123",
			want:  "test123",
		},
		{
			name:  "special characters unchanged",
			input: "Test!@#$",
			want:  "test!@#$",
		},
		{
			name:  "URL",
			input: "HTTP://LOCALHOST:8080",
			want:  "http://localhost:8080",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLowerASCII(tt.input)
			if got != tt.want {
				t.Errorf("toLowerASCII(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClientType_String(t *testing.T) {
	tests := []struct {
		name       string
		clientType ClientType
		want       string
	}{
		{
			name:       "TextClient",
			clientType: TextClient,
			want:       "text",
		},
		{
			name:       "ImageClient",
			clientType: ImageClient,
			want:       "image",
		},
		{
			name:       "VisionClient",
			clientType: VisionClient,
			want:       "vision",
		},
		{
			name:       "unknown type",
			clientType: ClientType(99),
			want:       "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.clientType.String()
			if got != tt.want {
				t.Errorf("ClientType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAIClientConfig(t *testing.T) {
	// Test that AIClientConfig can be created with all fields
	config := AIClientConfig{
		APIKey:      "test-key",
		BaseURL:     "https://api.example.com/v1",
		FallbackURL: "https://fallback.example.com/v1",
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		Timeout:     60 * time.Second,
	}

	if config.APIKey != "test-key" {
		t.Errorf("AIClientConfig.APIKey = %q, want %q", config.APIKey, "test-key")
	}
	if config.BaseURL != "https://api.example.com/v1" {
		t.Errorf("AIClientConfig.BaseURL = %q, want %q", config.BaseURL, "https://api.example.com/v1")
	}
	if config.FallbackURL != "https://fallback.example.com/v1" {
		t.Errorf("AIClientConfig.FallbackURL = %q, want %q", config.FallbackURL, "https://fallback.example.com/v1")
	}
	if config.HTTPClient == nil {
		t.Error("AIClientConfig.HTTPClient should not be nil")
	}
	if config.Timeout != 60*time.Second {
		t.Errorf("AIClientConfig.Timeout = %v, want %v", config.Timeout, 60*time.Second)
	}
}
