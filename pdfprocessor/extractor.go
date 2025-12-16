// Package pdfprocessor provides PDF processing functionality for CanvusLocalLLM.
//
// extractor.go implements the Extractor molecule that extracts text from PDF files.
// It uses the ledongthuc/pdf library for PDF parsing and composes:
//   - atoms.go: EstimateTokenCount for providing token estimates of extracted text
package pdfprocessor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ErrNoPDFContent is returned when a PDF contains no extractable text.
var ErrNoPDFContent = errors.New("no text content found in PDF")

// ErrEmptyPath is returned when an empty file path is provided.
var ErrEmptyPath = errors.New("empty PDF path provided")

// PageResult represents extracted text from a single PDF page.
type PageResult struct {
	// PageNumber is the 1-indexed page number
	PageNumber int

	// Text is the extracted text content
	Text string

	// EstimatedTokens is the estimated token count for this page
	EstimatedTokens int

	// Error is non-nil if extraction failed for this page
	Error error
}

// ExtractionResult contains the complete result of PDF text extraction.
type ExtractionResult struct {
	// Text is the full extracted text from all pages
	Text string

	// TotalPages is the number of pages in the PDF
	TotalPages int

	// ExtractedPages is the number of pages that yielded text
	ExtractedPages int

	// SkippedPages is the number of pages that were skipped (empty or error)
	SkippedPages int

	// EstimatedTokens is the estimated total token count
	EstimatedTokens int

	// Pages contains per-page extraction results
	Pages []PageResult

	// Errors contains any errors encountered during extraction
	Errors []error
}

// ExtractorConfig holds configuration for PDF text extraction.
type ExtractorConfig struct {
	// SkipEmptyPages when true excludes pages with no text from results
	SkipEmptyPages bool

	// PageSeparator is the string inserted between page texts
	// Defaults to "\n\n" if empty
	PageSeparator string

	// ContinueOnError when true continues extraction even if some pages fail
	ContinueOnError bool

	// MaxPages limits extraction to first N pages (0 for all pages)
	MaxPages int
}

// DefaultExtractorConfig returns sensible default configuration.
func DefaultExtractorConfig() ExtractorConfig {
	return ExtractorConfig{
		SkipEmptyPages:  true,
		PageSeparator:   "\n\n",
		ContinueOnError: true,
		MaxPages:        0,
	}
}

// Extractor extracts text from PDF files.
type Extractor struct {
	config ExtractorConfig
}

// NewExtractor creates a new Extractor with the given configuration.
func NewExtractor(config ExtractorConfig) *Extractor {
	// Apply defaults for empty separator
	if config.PageSeparator == "" {
		config.PageSeparator = "\n\n"
	}
	return &Extractor{config: config}
}

// NewDefaultExtractor creates an Extractor with default configuration.
func NewDefaultExtractor() *Extractor {
	return NewExtractor(DefaultExtractorConfig())
}

// Extract extracts text from a PDF file at the given path.
// It returns an ExtractionResult containing the extracted text and metadata.
//
// Example:
//
//	extractor := NewDefaultExtractor()
//	result, err := extractor.Extract("/path/to/document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Text)
func (e *Extractor) Extract(pdfPath string) (*ExtractionResult, error) {
	if pdfPath == "" {
		return nil, ErrEmptyPath
	}

	// Open PDF file
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	return e.extractFromReader(r)
}

// ExtractFromReader extracts text from a PDF reader.
// This is useful when the PDF is already loaded or comes from a non-file source.
//
// Example:
//
//	f, r, err := pdf.Open("document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//	extractor := NewDefaultExtractor()
//	result, err := extractor.ExtractFromReader(r)
func (e *Extractor) ExtractFromReader(r *pdf.Reader) (*ExtractionResult, error) {
	if r == nil {
		return nil, errors.New("nil PDF reader provided")
	}
	return e.extractFromReader(r)
}

// extractFromReader performs the actual extraction from a pdf.Reader.
func (e *Extractor) extractFromReader(r *pdf.Reader) (*ExtractionResult, error) {
	totalPages := r.NumPage()

	result := &ExtractionResult{
		TotalPages: totalPages,
		Pages:      make([]PageResult, 0, totalPages),
		Errors:     make([]error, 0),
	}

	var textBuilder strings.Builder

	// Determine how many pages to process
	pagesToProcess := totalPages
	if e.config.MaxPages > 0 && e.config.MaxPages < totalPages {
		pagesToProcess = e.config.MaxPages
	}

	// Extract text from each page (1-indexed in ledongthuc/pdf)
	for pageIndex := 1; pageIndex <= pagesToProcess; pageIndex++ {
		pageResult := e.extractPage(r, pageIndex)
		result.Pages = append(result.Pages, pageResult)

		if pageResult.Error != nil {
			result.Errors = append(result.Errors, fmt.Errorf("page %d: %w", pageIndex, pageResult.Error))
			result.SkippedPages++

			if !e.config.ContinueOnError {
				return result, pageResult.Error
			}
			continue
		}

		if pageResult.Text == "" {
			result.SkippedPages++
			continue
		}

		result.ExtractedPages++

		// Add page separator if not first page with content
		if textBuilder.Len() > 0 {
			textBuilder.WriteString(e.config.PageSeparator)
		}
		textBuilder.WriteString(pageResult.Text)
	}

	result.Text = textBuilder.String()
	result.EstimatedTokens = EstimateTokenCount(result.Text)

	// Return error if no content was extracted
	if result.Text == "" {
		return result, ErrNoPDFContent
	}

	return result, nil
}

// extractPage extracts text from a single page.
func (e *Extractor) extractPage(r *pdf.Reader, pageIndex int) PageResult {
	result := PageResult{
		PageNumber: pageIndex,
	}

	p := r.Page(pageIndex)
	if p.V.IsNull() {
		// Empty page - not an error, just no content
		return result
	}

	text, err := p.GetPlainText(nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract text: %w", err)
		return result
	}

	// Trim whitespace from extracted text
	result.Text = strings.TrimSpace(text)
	result.EstimatedTokens = EstimateTokenCount(result.Text)

	return result
}

// ExtractText is a convenience function that extracts text from a PDF file
// using default configuration and returns just the text content.
//
// Example:
//
//	text, err := ExtractText("/path/to/document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(text)
func ExtractText(pdfPath string) (string, error) {
	extractor := NewDefaultExtractor()
	result, err := extractor.Extract(pdfPath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ExtractTextWithTokenEstimate extracts text and returns both the text and
// estimated token count.
//
// Example:
//
//	text, tokens, err := ExtractTextWithTokenEstimate("/path/to/document.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Extracted %d tokens\n", tokens)
func ExtractTextWithTokenEstimate(pdfPath string) (string, int, error) {
	extractor := NewDefaultExtractor()
	result, err := extractor.Extract(pdfPath)
	if err != nil {
		return "", 0, err
	}
	return result.Text, result.EstimatedTokens, nil
}

