package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCanvasConnection(t *testing.T) {
	t.Log("\n=== Testing Canvas Connection ===")

	// First, make a basic request to the server root
	serverURL := os.Getenv("CANVUS_SERVER")
	t.Logf("Testing connection to server: %s", serverURL)

	// Create HTTP client with same TLS settings
	httpClient := client.HTTP

	// Try to connect to server root
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		t.Fatalf("❌ Failed to create request: %v", err)
	}

	req.Header.Set("Private-Token", os.Getenv("CANVUS_API_KEY"))
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("❌ Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Server response status: %s", resp.Status)

	// Now try to list available canvases
	canvasesURL := fmt.Sprintf("%s/api/v1/canvases", strings.TrimRight(serverURL, "/"))
	req, err = http.NewRequest("GET", canvasesURL, nil)
	if err != nil {
		t.Fatalf("❌ Failed to create canvases request: %v", err)
	}

	req.Header.Set("Private-Token", os.Getenv("CANVUS_API_KEY"))
	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatalf("❌ Failed to list canvases: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Canvases response status: %s", resp.Status)

	if resp.StatusCode == http.StatusOK {
		t.Log("✅ Successfully connected to Canvus server")

		// Read and parse the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("❌ Failed to read response body: %v", err)
		}

		var canvases []map[string]interface{}
		if err := json.Unmarshal(body, &canvases); err != nil {
			t.Fatalf("❌ Failed to parse canvases: %v", err)
		}

		t.Log("\nAvailable Canvases:")
		for _, canvas := range canvases {
			t.Logf("ID: %v, Name: %v", canvas["id"], canvas["name"])
		}

		t.Log("\nPlease update your .env file with one of these canvas IDs")
	} else {
		t.Errorf("❌ Failed to list canvases: %s", resp.Status)
	}
}
