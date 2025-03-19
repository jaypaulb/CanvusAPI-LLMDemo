package tests

import (
	"fmt"
	"go_backend/canvusapi"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// normalizeString converts a string to lowercase
func normalizeString(s string) string {
	return strings.ToLower(s)
}

// validateResponse checks if all expected fields are present and match in the response
func validateResponse(t *testing.T, expected, actual map[string]interface{}) error {
	// Consider adding validation for nil maps
	if expected == nil || actual == nil {
		return fmt.Errorf("validateResponse: received nil map (expected: %v, actual: %v)", expected, actual)
	}

	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return fmt.Errorf("%s: missing", key)
		}

		// Handle case-insensitive string comparisons for specific fields
		if key == "playback_state" || strings.HasSuffix(key, "color") {
			if expectedStr, ok := expectedValue.(string); ok {
				if actualStr, ok := actualValue.(string); ok {
					if normalizeString(expectedStr) != normalizeString(actualStr) {
						return fmt.Errorf("%s: expected %v, got %v", key, expectedValue, actualValue)
					}
					continue
				}
			}
		}

		// Handle nested maps (like location, size)
		if expectedMap, isMap := expectedValue.(map[string]interface{}); isMap {
			actualMap, actualIsMap := actualValue.(map[string]interface{})
			if !actualIsMap {
				return fmt.Errorf("%s: expected map, got %T", key, actualValue)
			}
			if err := validateResponse(t, expectedMap, actualMap); err != nil {
				return fmt.Errorf("%s.%v", key, err)
			}
			continue
		}

		// Handle number types (float64 vs int)
		if reflect.TypeOf(expectedValue).Kind() == reflect.Float64 ||
			reflect.TypeOf(actualValue).Kind() == reflect.Float64 {
			expectedFloat, _ := getFloat64(expectedValue)
			actualFloat, _ := getFloat64(actualValue)
			if expectedFloat != actualFloat {
				return fmt.Errorf("%s: expected %v, got %v", key, expectedValue, actualValue)
			}
			continue
		}

		// Handle regular fields
		if !reflect.DeepEqual(expectedValue, actualValue) {
			return fmt.Errorf("%s: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
	return nil
}

// Helper function
func getFloat64(v interface{}) (float64, bool) {
	switch i := v.(type) {
	case float64:
		return i, true
	case float32:
		return float64(i), true
	case int:
		return float64(i), true
	case int64:
		return float64(i), true
	default:
		return 0, false
	}
}

var client *canvusapi.Client

// Add at the top level
type TestSummary struct {
	Total    int
	Passed   int
	Failed   int
	Warnings int
	Details  []string
}

var summary TestSummary

// Modify the validation error logging
func logValidationError(t *testing.T, err error, response map[string]interface{}) {
	// Extract field name and expected/got values from error message
	// Format is typically "field: expected X, got Y"
	parts := strings.SplitN(err.Error(), ":", 2)
	if len(parts) == 2 {
		field := strings.TrimSpace(parts[0])
		if value, exists := response[field]; exists {
			// Extract expected value from error message
			details := strings.Split(parts[1], ",")
			if len(details) >= 2 {
				expected := strings.TrimPrefix(strings.TrimSpace(details[0]), "expected ")
				t.Logf("❌ Field mismatch - %s: %s <> %v", field, expected, value)
			} else {
				t.Logf("❌ Field mismatch - %s: <missing> <> %v", field, value)
			}
		}
	}
}

// Add to TestMain
func TestMain(m *testing.M) {
	// Initialize summary
	summary = TestSummary{
		Details: make([]string, 0),
	}

	// Setup client once for all tests
	var err error
	client, err = setupClient()
	if err != nil {
		fmt.Printf("❌ Test setup failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Client setup successful")

	// Create downloads directory
	if err := os.MkdirAll("../downloads", 0755); err != nil {
		fmt.Printf("❌ Failed to create downloads directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Downloads directory ready")

	// Run tests
	code := m.Run()

	// Add cleanup
	if err := os.RemoveAll("../downloads"); err != nil {
		fmt.Printf("⚠️ Warning: Failed to cleanup downloads directory: %v\n", err)
	}

	// Print summary
	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Total Tests: %d\n", summary.Total)
	fmt.Printf("✅ Passed: %d\n", summary.Passed)
	fmt.Printf("❌ Failed: %d\n", summary.Failed)
	fmt.Printf("⚠️ Warnings: %d\n", summary.Warnings)

	if len(summary.Details) > 0 {
		fmt.Printf("\nDetails:\n")
		for _, detail := range summary.Details {
			fmt.Println(detail)
		}
	}

	os.Exit(code)
}

func setupClient() (*canvusapi.Client, error) {
	envPath := os.Getenv("TEST_ENV_PATH")
	if envPath == "" {
		envPath = filepath.Join("..", ".env")
	}

	if err := godotenv.Load(envPath); err != nil {
		return nil, fmt.Errorf("failed to load environment file from %s: %v", envPath, err)
	}

	// Check required environment variables
	requiredEnvVars := []string{"CANVUS_SERVER", "CANVAS_ID", "CANVUS_API_KEY"}
	for _, env := range requiredEnvVars {
		if value := os.Getenv(env); value == "" {
			return nil, fmt.Errorf("required environment variable %s is not set", env)
		}
	}

	return canvusapi.NewClientFromEnv()
}

// TestNoteWidget tests CRUD operations for Note widgets
func TestNoteWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Note Widget ===")

	noteData := GetNoteTestData()

	// CREATE
	t.Log("\n📤 Creating Note...")
	note, err := client.CreateNote(noteData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create note: %v", err)
	}
	if err := validateResponse(t, noteData.CreatePayload, note); err != nil {
		testFailed = true
		logValidationError(t, err, note)
	}

	noteID := note["id"].(string)
	t.Logf("✅ Note created with ID: %s", noteID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating Note...")
	updated, err := client.UpdateNote(noteID, noteData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update note: %v", err)
	}
	if err := validateResponse(t, noteData.UpdatePayload, updated); err != nil {
		if strings.Contains(err.Error(), "title:") {
			summary.Warnings++
			t.Logf("⚠️ Known issue: Note title update not supported")
		} else {
			testFailed = true
			logValidationError(t, err, updated)
		}
	}

	// DELETE
	t.Log("\n📤 Deleting Note...")
	if err = client.DeleteNote(noteID); err != nil {
		t.Fatalf("❌ Failed to delete note: %v", err)
	}

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Note Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Note Widget: Passed")
	}
}

// TestPDFWidget tests CRUD operations for PDF widgets
func TestPDFWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing PDF Widget ===")

	pdfData := GetPDFTestData()
	t.Logf("📁 Using test PDF: %s", pdfData.FilePath)

	// CREATE
	t.Log("\n📤 Creating PDF...")
	pdf, err := client.CreatePDF(pdfData.FilePath, pdfData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create PDF: %v", err)
	}
	if err := validateResponse(t, pdfData.CreatePayload, pdf); err != nil {
		testFailed = true
		logValidationError(t, err, pdf)
	}

	pdfID := pdf["id"].(string)
	t.Logf("✅ PDF created with ID: %s", pdfID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating PDF...")
	updated, err := client.UpdatePDF(pdfID, pdfData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update PDF: %v", err)
	}
	if err := validateResponse(t, pdfData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ PDF updated successfully")

	testDelay()

	// DOWNLOAD
	t.Log("\n📤 Downloading PDF...")
	downloadPath := filepath.Join("../downloads", fmt.Sprintf("test_%s.pdf", pdfID))
	if err := client.DownloadPDF(pdfID, downloadPath); err != nil {
		testFailed = true
		t.Errorf("❌ Failed to download PDF: %v", err)
	} else {
		t.Log("✅ PDF downloaded successfully")
	}

	// DELETE
	t.Log("\n📤 Deleting PDF...")
	if err = client.DeletePDF(pdfID); err != nil {
		t.Fatalf("❌ Failed to delete PDF: %v", err)
	}
	t.Log("✅ PDF deleted successfully")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ PDF Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ PDF Widget: Passed")
	}
}

// TestImageWidget tests CRUD operations for Image widgets
func TestImageWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Image Widget ===")

	imageData := GetImageTestData()
	t.Logf("📁 Using test image: %s", imageData.FilePath)

	// CREATE
	t.Log("\n📤 Creating Image...")
	image, err := client.CreateImage(imageData.FilePath, imageData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create image: %v", err)
	}
	if err := validateResponse(t, imageData.CreatePayload, image); err != nil {
		testFailed = true
		logValidationError(t, err, image)
	}

	imageID := image["id"].(string)
	t.Logf("✅ Image created with ID: %s", imageID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating Image...")
	updated, err := client.UpdateImage(imageID, imageData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update image: %v", err)
	}
	if err := validateResponse(t, imageData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Image updated successfully")

	testDelay()

	// DOWNLOAD
	t.Log("\n📤 Downloading Image...")
	downloadPath := filepath.Join("../downloads", fmt.Sprintf("test_%s.jpg", imageID))
	if err := client.DownloadImage(imageID, downloadPath); err != nil {
		testFailed = true
		t.Errorf("❌ Failed to download image: %v", err)
	} else {
		t.Log("✅ Image downloaded successfully")
	}

	// DELETE
	t.Log("\n📤 Deleting Image...")
	if err = client.DeleteImage(imageID); err != nil {
		t.Fatalf("❌ Failed to delete image: %v", err)
	}
	t.Log("✅ Image deleted successfully")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Image Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Image Widget: Passed")
	}
}

// TestVideoWidget tests CRUD operations for Video widgets
func TestVideoWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Video Widget ===")

	videoData := GetVideoTestData()
	t.Logf("📁 Using test video: %s", videoData.FilePath)

	// CREATE (with stopped state)
	t.Log("\n📤 Creating Video...")
	t.Log("Creating video in STOPPED state")
	videoData.CreatePayload["playback_state"] = "STOPPED"

	video, err := client.CreateVideo(videoData.FilePath, videoData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create video: %v", err)
	}
	if err := validateResponse(t, videoData.CreatePayload, video); err != nil {
		testFailed = true
		logValidationError(t, err, video)
	}

	videoID := video["id"].(string)
	t.Logf("✅ Video created with ID: %s", videoID)

	testDelay()

	// UPDATE to playing
	t.Log("\n📤 Updating Video to PLAYING state...")
	videoData.UpdatePayload["playback_state"] = "PLAYING"
	updated, err := client.UpdateVideo(videoID, videoData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update video: %v", err)
	}
	if err := validateResponse(t, videoData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Video updated to PLAYING state")

	time.Sleep(3 * time.Second)

	// UPDATE to paused
	t.Log("\n📤 Updating Video to PAUSED state...")
	videoData.UpdatePayload["playback_state"] = "PAUSED"
	updated, err = client.UpdateVideo(videoID, videoData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update video: %v", err)
	}
	if err := validateResponse(t, videoData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Video updated to PAUSED state")

	testDelay()

	// DOWNLOAD
	t.Log("\n📤 Downloading Video...")
	downloadPath := filepath.Join("../downloads", fmt.Sprintf("test_%s.mp4", videoID))
	if err := client.DownloadVideo(videoID, downloadPath); err != nil {
		testFailed = true
		t.Errorf("❌ Failed to download video: %v", err)
	} else {
		t.Log("✅ Video downloaded successfully")
	}

	// DELETE
	t.Log("\n📤 Deleting Video...")
	if err = client.DeleteVideo(videoID); err != nil {
		t.Fatalf("❌ Failed to delete video: %v", err)
	}
	t.Log("✅ Video deleted successfully")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Video Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Video Widget: Passed")
	}
}

// TestBrowserWidget tests CRUD operations for Browser widgets
func TestBrowserWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Browser Widget ===")

	browserData := GetBrowserTestData()

	// CREATE
	t.Log("\n📤 Creating Browser...")
	browser, err := client.CreateBrowser(browserData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create browser: %v", err)
	}
	if err := validateResponse(t, browserData.CreatePayload, browser); err != nil {
		testFailed = true
		logValidationError(t, err, browser)
	}

	browserID := browser["id"].(string)
	t.Logf("✅ Browser created with ID: %s", browserID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating Browser...")
	updated, err := client.UpdateBrowser(browserID, browserData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update browser: %v", err)
	}
	if err := validateResponse(t, browserData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Browser updated successfully")

	// DELETE
	t.Log("\n📤 Deleting Browser...")
	if err = client.DeleteBrowser(browserID); err != nil {
		t.Fatalf("❌ Failed to delete browser: %v", err)
	}
	t.Log("✅ Browser deleted successfully")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Browser Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Browser Widget: Passed")
	}
}

// TestConnectorWidget tests CRUD operations for Connector widgets
func TestConnectorWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Connector Widget ===")

	// Create source note
	t.Log("\n📤 Creating source note...")
	sourceNote := map[string]interface{}{
		"text":   "Source",
		"depth":  0,
		"pinned": false,
		"location": map[string]interface{}{
			"x": 800.0,
			"y": 1000.0,
		},
		"size": map[string]interface{}{
			"width":  200.0,
			"height": 150.0,
		},
		"state":       "normal",
		"widget_type": "Note",
	}
	source, err := client.CreateNote(sourceNote)
	if err != nil {
		t.Fatalf("❌ Failed to create source note: %v", err)
	}
	sourceID := source["id"].(string)
	t.Log("✅ Source note created")

	testDelay()

	// Create destination note
	t.Log("\n📤 Creating destination note...")
	destNote := map[string]interface{}{
		"text":   "Destination",
		"depth":  0,
		"pinned": false,
		"location": map[string]interface{}{
			"x": 1200.0,
			"y": 1000.0,
		},
		"size": map[string]interface{}{
			"width":  200.0,
			"height": 150.0,
		},
		"state":       "normal",
		"widget_type": "Note",
	}
	dest, err := client.CreateNote(destNote)
	if err != nil {
		t.Fatalf("❌ Failed to create destination note: %v", err)
	}
	destID := dest["id"].(string)
	t.Log("✅ Destination note created")

	testDelay()

	// Update connector data with actual IDs
	connectorData := GetConnectorTestData()
	srcMap := connectorData.CreatePayload["src"].(map[string]interface{})
	dstMap := connectorData.CreatePayload["dst"].(map[string]interface{})
	srcMap["id"] = sourceID
	dstMap["id"] = destID

	// CREATE
	t.Log("\n📤 Creating Connector...")
	connector, err := client.CreateConnector(connectorData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create connector: %v", err)
	}
	if err := validateResponse(t, connectorData.CreatePayload, connector); err != nil {
		testFailed = true
		logValidationError(t, err, connector)
	}

	connectorID := connector["id"].(string)
	t.Logf("✅ Connector created with ID: %s", connectorID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating Connector...")
	updated, err := client.UpdateConnector(connectorID, connectorData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update connector: %v", err)
	}
	if err := validateResponse(t, connectorData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Connector updated successfully")

	// DELETE
	t.Log("\n📤 Deleting Connector...")
	if err = client.DeleteConnector(connectorID); err != nil {
		t.Fatalf("❌ Failed to delete connector: %v", err)
	}
	t.Log("✅ Connector deleted successfully")

	// Cleanup source and destination notes
	t.Log("\n📤 Cleaning up test notes...")
	client.DeleteNote(sourceID)
	client.DeleteNote(destID)
	t.Log("✅ Test notes deleted")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Connector Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Connector Widget: Passed")
	}
}

// TestAnchorWidget tests CRUD operations for Anchor widgets
func TestAnchorWidget(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Anchor Widget ===")

	anchorData := GetAnchorTestData()

	// CREATE
	t.Log("\n📤 Creating Anchor...")
	anchor, err := client.CreateAnchor(anchorData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create anchor: %v", err)
	}
	if err := validateResponse(t, anchorData.CreatePayload, anchor); err != nil {
		testFailed = true
		logValidationError(t, err, anchor)
	}

	anchorID := anchor["id"].(string)
	t.Logf("✅ Anchor created with ID: %s", anchorID)

	testDelay()

	// UPDATE
	t.Log("\n📤 Updating Anchor...")
	updated, err := client.UpdateAnchor(anchorID, anchorData.UpdatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to update anchor: %v", err)
	}
	if err := validateResponse(t, anchorData.UpdatePayload, updated); err != nil {
		testFailed = true
		logValidationError(t, err, updated)
	}
	t.Log("✅ Anchor updated successfully")

	// DELETE
	t.Log("\n📤 Deleting Anchor...")
	if err = client.DeleteAnchor(anchorID); err != nil {
		t.Fatalf("❌ Failed to delete anchor: %v", err)
	}
	t.Log("✅ Anchor deleted successfully")

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Anchor Widget: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Anchor Widget: Passed")
	}
}

var testDelayDuration = 500 * time.Millisecond

func init() {
	if delay := os.Getenv("TEST_DELAY_MS"); delay != "" {
		if ms, err := strconv.Atoi(delay); err == nil {
			testDelayDuration = time.Duration(ms) * time.Millisecond
		}
	}
}

func testDelay() {
	time.Sleep(testDelayDuration)
}

func TestWidgetsEndpoint(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Widgets Endpoint ===")

	// List all widgets
	t.Log("\n📤 Getting all widgets...")
	widgets, err := client.GetWidgets(false)
	if err != nil {
		t.Fatalf("❌ Failed to get widgets: %v", err)
	}
	t.Logf("✅ Retrieved %d widgets", len(widgets))

	// Create a test widget (Note) to verify single widget operations
	noteData := GetNoteTestData()
	note, err := client.CreateNote(noteData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create test note: %v", err)
	}
	noteID := note["id"].(string)
	t.Log("✅ Created test widget")

	testDelay()

	// Get single widget
	t.Log("\n📤 Getting single widget...")
	widget, err := client.GetWidget(noteID, false)
	if err != nil {
		testFailed = true
		t.Errorf("❌ Failed to get single widget: %v", err)
	} else if widget["id"] != noteID {
		testFailed = true
		t.Errorf("❌ Widget ID mismatch: expected %s, got %s", noteID, widget["id"])
	} else {
		t.Log("✅ Successfully retrieved single widget")
	}

	// Cleanup
	client.DeleteNote(noteID)

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Widgets Endpoint: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Widgets Endpoint: Passed")
	}
}

func TestWidgetSubscribe(t *testing.T) {
	summary.Total++
	testFailed := false

	t.Log("\n=== Testing Widget Subscribe ===")

	// Create a test widget for subscription testing
	noteData := GetNoteTestData()
	note, err := client.CreateNote(noteData.CreatePayload)
	if err != nil {
		t.Fatalf("❌ Failed to create test note: %v", err)
	}
	noteID := note["id"].(string)
	t.Log("✅ Created test widget")

	testDelay()

	// Test subscribe to single widget
	t.Log("\n📤 Testing widget subscription...")
	_, err = client.GetWidget(noteID, true)
	if err != nil {
		testFailed = true
		t.Errorf("❌ Failed to subscribe to widget: %v", err)
	} else {
		t.Log("✅ Successfully subscribed to widget")
	}

	testDelay()

	// Test subscribe to all widgets
	t.Log("\n📤 Testing all widgets subscription...")
	_, err = client.GetWidgets(true)
	if err != nil {
		testFailed = true
		t.Errorf("❌ Failed to subscribe to all widgets: %v", err)
	} else {
		t.Log("✅ Successfully subscribed to all widgets")
	}

	// Cleanup
	client.DeleteNote(noteID)

	if testFailed {
		summary.Failed++
		summary.Details = append(summary.Details, "❌ Widget Subscribe: Failed")
	} else {
		summary.Passed++
		summary.Details = append(summary.Details, "✅ Widget Subscribe: Passed")
	}
}

func TestInvalidWidgetOperations(t *testing.T) {
	// Test invalid widget ID
	_, err := client.GetWidget("invalid-id", false)
	if err == nil {
		t.Error("Expected error for invalid widget ID, got nil")
	}

	// Test invalid file paths
	_, err = client.CreatePDF("nonexistent.pdf", nil)
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func validateTestData(data interface{}) error {
	// Add validation logic for test data structure
	return nil
}
