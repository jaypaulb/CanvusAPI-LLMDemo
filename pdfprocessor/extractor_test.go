package pdfprocessor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// getTestPDFPath returns the path to the test PDF file.
// It checks both relative and absolute paths to work in different test contexts.
func getTestPDFPath() string {
	// Try relative path from pdfprocessor package
	relativePath := "../test_files/test_pdf.pdf"
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// Try absolute path
	homeDir, _ := os.UserHomeDir()
	absolutePath := filepath.Join(homeDir, "Projects/gh/CanvusLocalLLM/test_files/test_pdf.pdf")
	if _, err := os.Stat(absolutePath); err == nil {
		return absolutePath
	}

	// Return relative path and let test handle the error
	return relativePath
}

func TestNewExtractor(t *testing.T) {
	tests := []struct {
		name           string
		config         ExtractorConfig
		wantSeparator  string
	}{
		{
			name:           "default config",
			config:         DefaultExtractorConfig(),
			wantSeparator:  "\n\n",
		},
		{
			name: "custom separator",
			config: ExtractorConfig{
				PageSeparator: "---PAGE---",
			},
			wantSeparator: "---PAGE---",
		},
		{
			name: "empty separator gets default",
			config: ExtractorConfig{
				PageSeparator: "",
			},
			wantSeparator: "\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExtractor(tt.config)
			if e == nil {
				t.Fatal("NewExtractor returned nil")
			}
			if e.config.PageSeparator != tt.wantSeparator {
				t.Errorf("PageSeparator = %q, want %q", e.config.PageSeparator, tt.wantSeparator)
			}
		})
	}
}

func TestNewDefaultExtractor(t *testing.T) {
	e := NewDefaultExtractor()
	if e == nil {
		t.Fatal("NewDefaultExtractor returned nil")
	}

	// Verify default config values
	if e.config.SkipEmptyPages != true {
		t.Error("SkipEmptyPages should be true by default")
	}
	if e.config.PageSeparator != "\n\n" {
		t.Errorf("PageSeparator = %q, want %q", e.config.PageSeparator, "\n\n")
	}
	if e.config.ContinueOnError != true {
		t.Error("ContinueOnError should be true by default")
	}
	if e.config.MaxPages != 0 {
		t.Errorf("MaxPages = %d, want 0", e.config.MaxPages)
	}
}

func TestExtractor_Extract_EmptyPath(t *testing.T) {
	e := NewDefaultExtractor()
	_, err := e.Extract("")
	if err != ErrEmptyPath {
		t.Errorf("Extract(\"\") error = %v, want ErrEmptyPath", err)
	}
}

func TestExtractor_Extract_NonexistentFile(t *testing.T) {
	e := NewDefaultExtractor()
	_, err := e.Extract("/nonexistent/path/to/file.pdf")
	if err == nil {
		t.Error("Extract with nonexistent file should return error")
	}
	if !strings.Contains(err.Error(), "failed to open PDF") {
		t.Errorf("error message should contain 'failed to open PDF', got: %v", err)
	}
}

func TestExtractor_Extract_ValidPDF(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	e := NewDefaultExtractor()
	result, err := e.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("result is nil")
	}

	if result.TotalPages <= 0 {
		t.Error("TotalPages should be > 0")
	}

	if result.Text == "" {
		t.Error("Text should not be empty")
	}

	if result.EstimatedTokens <= 0 {
		t.Error("EstimatedTokens should be > 0")
	}

	if len(result.Pages) != result.TotalPages {
		t.Errorf("Pages length = %d, want %d", len(result.Pages), result.TotalPages)
	}

	// Verify page results have correct page numbers
	for i, page := range result.Pages {
		if page.PageNumber != i+1 {
			t.Errorf("Page %d has PageNumber = %d", i, page.PageNumber)
		}
	}
}

func TestExtractor_Extract_WithMaxPages(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	config := DefaultExtractorConfig()
	config.MaxPages = 1
	e := NewExtractor(config)

	result, err := e.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(result.Pages) > 1 {
		t.Errorf("Should only extract 1 page, got %d", len(result.Pages))
	}
}

func TestExtractor_Extract_CustomSeparator(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	// First check if PDF has multiple pages with content
	defaultExtractor := NewDefaultExtractor()
	defaultResult, err := defaultExtractor.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if defaultResult.ExtractedPages < 2 {
		t.Skip("Test PDF has fewer than 2 pages with content, skipping separator test")
	}

	// Now test with custom separator
	config := DefaultExtractorConfig()
	config.PageSeparator = "<<<PAGE>>>"
	e := NewExtractor(config)

	result, err := e.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !strings.Contains(result.Text, "<<<PAGE>>>") {
		t.Error("Result should contain custom page separator")
	}
}

func TestExtractText(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	text, err := ExtractText(pdfPath)
	if err != nil {
		t.Fatalf("ExtractText failed: %v", err)
	}

	if text == "" {
		t.Error("ExtractText should return non-empty text")
	}
}

func TestExtractText_EmptyPath(t *testing.T) {
	_, err := ExtractText("")
	if err != ErrEmptyPath {
		t.Errorf("ExtractText(\"\") error = %v, want ErrEmptyPath", err)
	}
}

func TestExtractTextWithTokenEstimate(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	text, tokens, err := ExtractTextWithTokenEstimate(pdfPath)
	if err != nil {
		t.Fatalf("ExtractTextWithTokenEstimate failed: %v", err)
	}

	if text == "" {
		t.Error("text should not be empty")
	}

	if tokens <= 0 {
		t.Error("tokens should be > 0")
	}

	// Verify token estimate is consistent with EstimateTokenCount
	expectedTokens := EstimateTokenCount(text)
	if tokens != expectedTokens {
		t.Errorf("tokens = %d, want %d", tokens, expectedTokens)
	}
}

func TestExtractor_ExtractFromReader_Nil(t *testing.T) {
	e := NewDefaultExtractor()
	_, err := e.ExtractFromReader(nil)
	if err == nil {
		t.Error("ExtractFromReader(nil) should return error")
	}
}

func TestExtractionResult_Consistency(t *testing.T) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Test PDF file not found, skipping integration test")
	}

	e := NewDefaultExtractor()
	result, err := e.Extract(pdfPath)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify ExtractedPages + SkippedPages == TotalPages
	if result.ExtractedPages+result.SkippedPages != result.TotalPages {
		t.Errorf("ExtractedPages(%d) + SkippedPages(%d) != TotalPages(%d)",
			result.ExtractedPages, result.SkippedPages, result.TotalPages)
	}

	// Verify EstimatedTokens is sum of page tokens (approximately)
	// Note: Due to separator tokens, total may be slightly higher than sum
	sumPageTokens := 0
	for _, page := range result.Pages {
		sumPageTokens += page.EstimatedTokens
	}

	// Allow for separator tokens to account for difference
	separatorTokens := EstimateTokenCount(e.config.PageSeparator) * (result.ExtractedPages - 1)
	if separatorTokens < 0 {
		separatorTokens = 0
	}
	expectedTotal := sumPageTokens + separatorTokens

	// Allow small tolerance for rounding differences
	tolerance := 5
	diff := result.EstimatedTokens - expectedTotal
	if diff < -tolerance || diff > tolerance {
		t.Errorf("EstimatedTokens(%d) doesn't match expected(%d) within tolerance(%d)",
			result.EstimatedTokens, expectedTotal, tolerance)
	}
}

func TestDefaultExtractorConfig(t *testing.T) {
	config := DefaultExtractorConfig()

	if config.SkipEmptyPages != true {
		t.Error("SkipEmptyPages should default to true")
	}
	if config.PageSeparator != "\n\n" {
		t.Errorf("PageSeparator should default to '\\n\\n', got %q", config.PageSeparator)
	}
	if config.ContinueOnError != true {
		t.Error("ContinueOnError should default to true")
	}
	if config.MaxPages != 0 {
		t.Errorf("MaxPages should default to 0, got %d", config.MaxPages)
	}
}

// Benchmark tests

func BenchmarkExtractText(b *testing.B) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		b.Skip("Test PDF file not found, skipping benchmark")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractText(pdfPath)
	}
}

func BenchmarkExtractor_Extract(b *testing.B) {
	pdfPath := getTestPDFPath()
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		b.Skip("Test PDF file not found, skipping benchmark")
	}

	e := NewDefaultExtractor()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Extract(pdfPath)
	}
}
