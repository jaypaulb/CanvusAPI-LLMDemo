package imagegen

import "testing"

func TestIsAzureEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "empty string returns false",
			endpoint: "",
			expected: false,
		},
		{
			name:     "openai.azure.com returns true",
			endpoint: "https://myresource.openai.azure.com",
			expected: true,
		},
		{
			name:     "openai.azure.com with path returns true",
			endpoint: "https://myresource.openai.azure.com/openai/deployments/gpt-4",
			expected: true,
		},
		{
			name:     "cognitiveservices.azure.com returns true",
			endpoint: "https://myresource.cognitiveservices.azure.com",
			expected: true,
		},
		{
			name:     "case insensitive - uppercase",
			endpoint: "https://myresource.OPENAI.AZURE.COM",
			expected: true,
		},
		{
			name:     "case insensitive - mixed case",
			endpoint: "https://myresource.OpenAI.Azure.COM",
			expected: true,
		},
		{
			name:     "standard OpenAI returns false",
			endpoint: "https://api.openai.com/v1",
			expected: false,
		},
		{
			name:     "localhost returns false",
			endpoint: "http://localhost:1234",
			expected: false,
		},
		{
			name:     "arbitrary URL returns false",
			endpoint: "https://example.com/api",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAzureEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("IsAzureEndpoint(%q) = %v, want %v", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestIsOpenAIEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "empty string returns false",
			endpoint: "",
			expected: false,
		},
		{
			name:     "api.openai.com returns true",
			endpoint: "https://api.openai.com",
			expected: true,
		},
		{
			name:     "api.openai.com with path returns true",
			endpoint: "https://api.openai.com/v1/chat/completions",
			expected: true,
		},
		{
			name:     "case insensitive",
			endpoint: "https://API.OPENAI.COM/v1",
			expected: true,
		},
		{
			name:     "Azure endpoint returns false",
			endpoint: "https://myresource.openai.azure.com",
			expected: false,
		},
		{
			name:     "localhost returns false",
			endpoint: "http://localhost:1234",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsOpenAIEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("IsOpenAIEndpoint(%q) = %v, want %v", tt.endpoint, result, tt.expected)
			}
		})
	}
}

func TestIsLocalEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "empty string returns false",
			endpoint: "",
			expected: false,
		},
		{
			name:     "localhost returns true",
			endpoint: "http://localhost:1234",
			expected: true,
		},
		{
			name:     "127.0.0.1 returns true",
			endpoint: "http://127.0.0.1:8080",
			expected: true,
		},
		{
			name:     "0.0.0.0 returns true",
			endpoint: "http://0.0.0.0:5000",
			expected: true,
		},
		{
			name:     "192.168.x.x returns true",
			endpoint: "http://192.168.1.100:5000",
			expected: true,
		},
		{
			name:     "10.x.x.x returns true",
			endpoint: "http://10.0.0.50:8080",
			expected: true,
		},
		{
			name:     "case insensitive - LOCALHOST",
			endpoint: "http://LOCALHOST:1234",
			expected: true,
		},
		{
			name:     "public IP returns false",
			endpoint: "http://203.0.113.50:8080",
			expected: false,
		},
		{
			name:     "api.openai.com returns false",
			endpoint: "https://api.openai.com",
			expected: false,
		},
		{
			name:     "Azure endpoint returns false",
			endpoint: "https://myresource.openai.azure.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("IsLocalEndpoint(%q) = %v, want %v", tt.endpoint, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkIsAzureEndpoint(b *testing.B) {
	endpoint := "https://myresource.openai.azure.com/openai/deployments/gpt-4"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsAzureEndpoint(endpoint)
	}
}

func BenchmarkIsOpenAIEndpoint(b *testing.B) {
	endpoint := "https://api.openai.com/v1/chat/completions"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsOpenAIEndpoint(endpoint)
	}
}

func BenchmarkIsLocalEndpoint(b *testing.B) {
	endpoint := "http://192.168.1.100:5000/v1"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsLocalEndpoint(endpoint)
	}
}
