package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"go_backend/canvusapi"
)

func main() {
	// Test with SSL validation enabled (default)
	fmt.Println("\n=== Testing with SSL validation enabled (default) ===")
	os.Setenv("ALLOW_SELF_SIGNED_CERTS", "false")
	testConnection()

	// Test with SSL validation disabled
	fmt.Println("\n=== Testing with SSL validation disabled ===")
	os.Setenv("ALLOW_SELF_SIGNED_CERTS", "true")
	testConnection()
}

func testConnection() {
	client, err := canvusapi.NewClientFromEnv()
	if err != nil {
		log.Printf("❌ Failed to create client: %v", err)
		return
	}

	// Test basic API call
	fmt.Println("Testing connection to server...")
	start := time.Now()
	info, err := client.GetCanvasInfo()
	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Connection failed: %v", err)
		return
	}

	fmt.Printf("✅ Connection successful!\n")
	fmt.Printf("⏱️ Response time: %v\n", duration)
	fmt.Printf("📦 Response data: %+v\n", info)
}
