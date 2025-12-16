// Package handlers provides the NoteUpdater molecule for note operations.
package handlers

import (
	"errors"
	"fmt"
	"time"
)

// NoteUpdater errors
var (
	// ErrMaxRetriesExceeded is returned when all retry attempts are exhausted.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	// ErrNilClient is returned when the API client is nil.
	ErrNilClient = errors.New("nil API client")
	// ErrEmptyNoteID is returned when the note ID is empty.
	ErrEmptyNoteID = errors.New("empty note ID")
)

// NoteAPIClient is the interface for note API operations.
// This allows for dependency injection and testing.
type NoteAPIClient interface {
	UpdateNote(id string, payload map[string]interface{}) (map[string]interface{}, error)
	UpdateImage(id string, payload map[string]interface{}) (map[string]interface{}, error)
}

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultRetryConfig returns sensible defaults for retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
	}
}

// NoteUpdater provides note update operations with retry logic and validation.
// It composes validation and location atoms for comprehensive note management.
//
// This is a molecule that composes atoms:
// - Validation atoms (ValidateUpdate, ValidateNonEmptyString)
// - Location atoms (ExtractLocation, ExtractSize)
//
// Example:
//
//	updater := handlers.NewNoteUpdater(client, handlers.DefaultRetryConfig())
//	err := updater.UpdateText(noteID, "New content")
type NoteUpdater struct {
	client      NoteAPIClient
	retryConfig RetryConfig
}

// NewNoteUpdater creates a new NoteUpdater with the given client and retry configuration.
func NewNoteUpdater(client NoteAPIClient, config RetryConfig) *NoteUpdater {
	return &NoteUpdater{
		client:      client,
		retryConfig: config,
	}
}

// UpdateText updates a note's text content with retry logic.
// Returns nil on success, or an error after all retries are exhausted.
//
// Example:
//
//	err := updater.UpdateText("note-123", "Updated text content")
func (u *NoteUpdater) UpdateText(noteID string, text string) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"text": text,
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdatePayload updates a note with an arbitrary payload map with retry logic.
// Use this for complex updates that need multiple fields.
//
// Example:
//
//	err := updater.UpdatePayload("note-123", map[string]interface{}{
//	    "text": "New text",
//	    "background_color": "#FF0000",
//	})
func (u *NoteUpdater) UpdatePayload(noteID string, payload map[string]interface{}) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}
	if payload == nil {
		return errors.New("nil payload")
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateLocation updates a note's location with retry logic.
// Uses the Location atom type for type safety.
//
// Example:
//
//	err := updater.UpdateLocation("note-123", handlers.Location{X: 100, Y: 200})
func (u *NoteUpdater) UpdateLocation(noteID string, loc Location) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"location": LocationToMap(loc),
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateSize updates a note's size with retry logic.
// Uses the NoteSize atom type for type safety.
//
// Example:
//
//	err := updater.UpdateSize("note-123", handlers.NoteSize{Width: 400, Height: 300})
func (u *NoteUpdater) UpdateSize(noteID string, size NoteSize) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"size": SizeToMap(size),
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateTextWithSize updates both text and size atomically with retry logic.
// This is useful when the note size needs to be recalculated based on content.
//
// Example:
//
//	content := "Long text content..."
//	size, _ := handlers.CalculateNoteSize(content, origWidth, origHeight, origScale)
//	err := updater.UpdateTextWithSize("note-123", content, size)
func (u *NoteUpdater) UpdateTextWithSize(noteID string, text string, size NoteSize) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"text": text,
		"size": SizeToMap(size),
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateBackgroundColor updates a note's background color with retry logic.
//
// Example:
//
//	newColor := handlers.ReduceBackgroundOpacity("#FF0000FF")
//	err := updater.UpdateBackgroundColor("note-123", newColor)
func (u *NoteUpdater) UpdateBackgroundColor(noteID string, color string) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"background_color": color,
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateScale updates a note's scale with retry logic.
//
// Example:
//
//	err := updater.UpdateScale("note-123", 0.5)
func (u *NoteUpdater) UpdateScale(noteID string, scale float64) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if noteID == "" {
		return ErrEmptyNoteID
	}

	payload := map[string]interface{}{
		"scale": scale,
	}

	return u.updateWithRetry(noteID, payload)
}

// UpdateImage updates an image widget with retry logic.
// This is similar to UpdatePayload but uses the image endpoint.
//
// Example:
//
//	err := updater.UpdateImage("image-123", map[string]interface{}{
//	    "title": "New title",
//	})
func (u *NoteUpdater) UpdateImage(imageID string, payload map[string]interface{}) error {
	if err := u.validateClient(); err != nil {
		return err
	}
	if imageID == "" {
		return ErrEmptyNoteID
	}
	if payload == nil {
		return errors.New("nil payload")
	}

	return u.updateImageWithRetry(imageID, payload)
}

// updateWithRetry performs the actual retry loop for note updates.
func (u *NoteUpdater) updateWithRetry(noteID string, payload map[string]interface{}) error {
	var lastErr error

	for attempt := 1; attempt <= u.retryConfig.MaxRetries; attempt++ {
		_, err := u.client.UpdateNote(noteID, payload)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < u.retryConfig.MaxRetries {
			time.Sleep(u.retryConfig.RetryDelay)
		}
	}

	return fmt.Errorf("%w: failed to update note %s after %d attempts: %v",
		ErrMaxRetriesExceeded, noteID, u.retryConfig.MaxRetries, lastErr)
}

// updateImageWithRetry performs the actual retry loop for image updates.
func (u *NoteUpdater) updateImageWithRetry(imageID string, payload map[string]interface{}) error {
	var lastErr error

	for attempt := 1; attempt <= u.retryConfig.MaxRetries; attempt++ {
		_, err := u.client.UpdateImage(imageID, payload)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't sleep after the last attempt
		if attempt < u.retryConfig.MaxRetries {
			time.Sleep(u.retryConfig.RetryDelay)
		}
	}

	return fmt.Errorf("%w: failed to update image %s after %d attempts: %v",
		ErrMaxRetriesExceeded, imageID, u.retryConfig.MaxRetries, lastErr)
}

// validateClient checks that the client is not nil.
func (u *NoteUpdater) validateClient() error {
	if u.client == nil {
		return ErrNilClient
	}
	return nil
}

// RetryableError wraps an error with retry attempt information.
// This can be used to track which errors are retryable.
type RetryableError struct {
	Err         error
	Attempt     int
	MaxAttempts int
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("attempt %d/%d: %v", e.Attempt, e.MaxAttempts, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}
