package handlers

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// mockNoteClient is a mock implementation of NoteAPIClient for testing.
type mockNoteClient struct {
	updateNoteCalls  int32
	updateImageCalls int32
	shouldFail       bool
	failUntil        int // Fail this many times before succeeding
	callCount        int32
	lastNoteID       string
	lastPayload      map[string]interface{}
}

func (m *mockNoteClient) UpdateNote(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	atomic.AddInt32(&m.updateNoteCalls, 1)
	atomic.AddInt32(&m.callCount, 1)
	m.lastNoteID = id
	m.lastPayload = payload

	if m.shouldFail {
		return nil, errors.New("mock error: update failed")
	}

	currentCall := atomic.LoadInt32(&m.callCount)
	if m.failUntil > 0 && int(currentCall) <= m.failUntil {
		return nil, errors.New("mock error: temporary failure")
	}

	return map[string]interface{}{"id": id}, nil
}

func (m *mockNoteClient) UpdateImage(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	atomic.AddInt32(&m.updateImageCalls, 1)
	atomic.AddInt32(&m.callCount, 1)
	m.lastNoteID = id
	m.lastPayload = payload

	if m.shouldFail {
		return nil, errors.New("mock error: update failed")
	}

	currentCall := atomic.LoadInt32(&m.callCount)
	if m.failUntil > 0 && int(currentCall) <= m.failUntil {
		return nil, errors.New("mock error: temporary failure")
	}

	return map[string]interface{}{"id": id}, nil
}

func TestNewNoteUpdater(t *testing.T) {
	client := &mockNoteClient{}
	config := DefaultRetryConfig()

	updater := NewNoteUpdater(client, config)

	if updater == nil {
		t.Fatal("NewNoteUpdater() returned nil")
	}
	if updater.client != client {
		t.Error("NewNoteUpdater() did not set client")
	}
	if updater.retryConfig.MaxRetries != config.MaxRetries {
		t.Errorf("NewNoteUpdater() MaxRetries = %d, want %d", updater.retryConfig.MaxRetries, config.MaxRetries)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries <= 0 {
		t.Errorf("DefaultRetryConfig().MaxRetries = %d, want > 0", config.MaxRetries)
	}
	if config.RetryDelay <= 0 {
		t.Errorf("DefaultRetryConfig().RetryDelay = %v, want > 0", config.RetryDelay)
	}
}

func TestNoteUpdater_UpdateText(t *testing.T) {
	tests := []struct {
		name      string
		noteID    string
		text      string
		clientNil bool
		wantErr   error
	}{
		{
			name:    "successful update",
			noteID:  "note-123",
			text:    "Hello world",
			wantErr: nil,
		},
		{
			name:    "empty note ID",
			noteID:  "",
			text:    "Hello",
			wantErr: ErrEmptyNoteID,
		},
		{
			name:      "nil client",
			noteID:    "note-123",
			text:      "Hello",
			clientNil: true,
			wantErr:   ErrNilClient,
		},
		{
			name:    "empty text is allowed",
			noteID:  "note-123",
			text:    "",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client NoteAPIClient
			if !tt.clientNil {
				client = &mockNoteClient{}
			}

			updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})
			err := updater.UpdateText(tt.noteID, tt.text)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("UpdateText() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("UpdateText() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateText() unexpected error = %v", err)
			}

			// Verify the mock received the call
			if !tt.clientNil {
				mock := client.(*mockNoteClient)
				if mock.lastNoteID != tt.noteID {
					t.Errorf("UpdateText() noteID = %s, want %s", mock.lastNoteID, tt.noteID)
				}
				if mock.lastPayload["text"] != tt.text {
					t.Errorf("UpdateText() payload text = %v, want %v", mock.lastPayload["text"], tt.text)
				}
			}
		})
	}
}

func TestNoteUpdater_UpdatePayload(t *testing.T) {
	tests := []struct {
		name    string
		noteID  string
		payload map[string]interface{}
		wantErr bool
	}{
		{
			name:    "successful update",
			noteID:  "note-123",
			payload: map[string]interface{}{"text": "hello", "scale": 0.5},
			wantErr: false,
		},
		{
			name:    "empty note ID",
			noteID:  "",
			payload: map[string]interface{}{"text": "hello"},
			wantErr: true,
		},
		{
			name:    "nil payload",
			noteID:  "note-123",
			payload: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockNoteClient{}
			updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

			err := updater.UpdatePayload(tt.noteID, tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdatePayload() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("UpdatePayload() unexpected error = %v", err)
			}
		})
	}
}

func TestNoteUpdater_UpdateLocation(t *testing.T) {
	client := &mockNoteClient{}
	updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

	loc := Location{X: 100, Y: 200}
	err := updater.UpdateLocation("note-123", loc)

	if err != nil {
		t.Errorf("UpdateLocation() unexpected error = %v", err)
	}

	// Verify payload structure
	locPayload, ok := client.lastPayload["location"].(map[string]float64)
	if !ok {
		t.Fatal("UpdateLocation() payload missing location field")
	}
	if locPayload["x"] != 100 {
		t.Errorf("UpdateLocation() x = %v, want 100", locPayload["x"])
	}
	if locPayload["y"] != 200 {
		t.Errorf("UpdateLocation() y = %v, want 200", locPayload["y"])
	}
}

func TestNoteUpdater_UpdateSize(t *testing.T) {
	client := &mockNoteClient{}
	updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

	size := NoteSize{Width: 400, Height: 300}
	err := updater.UpdateSize("note-123", size)

	if err != nil {
		t.Errorf("UpdateSize() unexpected error = %v", err)
	}

	// Verify payload structure
	sizePayload, ok := client.lastPayload["size"].(map[string]interface{})
	if !ok {
		t.Fatal("UpdateSize() payload missing size field")
	}
	if sizePayload["width"] != 400.0 {
		t.Errorf("UpdateSize() width = %v, want 400", sizePayload["width"])
	}
	if sizePayload["height"] != 300.0 {
		t.Errorf("UpdateSize() height = %v, want 300", sizePayload["height"])
	}
}

func TestNoteUpdater_UpdateTextWithSize(t *testing.T) {
	client := &mockNoteClient{}
	updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

	size := NoteSize{Width: 500, Height: 400}
	err := updater.UpdateTextWithSize("note-123", "Hello world", size)

	if err != nil {
		t.Errorf("UpdateTextWithSize() unexpected error = %v", err)
	}

	// Verify both text and size are in payload
	if client.lastPayload["text"] != "Hello world" {
		t.Errorf("UpdateTextWithSize() text = %v, want 'Hello world'", client.lastPayload["text"])
	}
	sizePayload, ok := client.lastPayload["size"].(map[string]interface{})
	if !ok {
		t.Fatal("UpdateTextWithSize() payload missing size field")
	}
	if sizePayload["width"] != 500.0 {
		t.Errorf("UpdateTextWithSize() width = %v, want 500", sizePayload["width"])
	}
}

func TestNoteUpdater_UpdateBackgroundColor(t *testing.T) {
	client := &mockNoteClient{}
	updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

	err := updater.UpdateBackgroundColor("note-123", "#FF0000FF")

	if err != nil {
		t.Errorf("UpdateBackgroundColor() unexpected error = %v", err)
	}

	if client.lastPayload["background_color"] != "#FF0000FF" {
		t.Errorf("UpdateBackgroundColor() color = %v, want '#FF0000FF'", client.lastPayload["background_color"])
	}
}

func TestNoteUpdater_UpdateScale(t *testing.T) {
	client := &mockNoteClient{}
	updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

	err := updater.UpdateScale("note-123", 0.5)

	if err != nil {
		t.Errorf("UpdateScale() unexpected error = %v", err)
	}

	if client.lastPayload["scale"] != 0.5 {
		t.Errorf("UpdateScale() scale = %v, want 0.5", client.lastPayload["scale"])
	}
}

func TestNoteUpdater_UpdateImage(t *testing.T) {
	tests := []struct {
		name    string
		imageID string
		payload map[string]interface{}
		wantErr bool
	}{
		{
			name:    "successful update",
			imageID: "image-123",
			payload: map[string]interface{}{"title": "test"},
			wantErr: false,
		},
		{
			name:    "empty image ID",
			imageID: "",
			payload: map[string]interface{}{"title": "test"},
			wantErr: true,
		},
		{
			name:    "nil payload",
			imageID: "image-123",
			payload: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockNoteClient{}
			updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

			err := updater.UpdateImage(tt.imageID, tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Error("UpdateImage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateImage() unexpected error = %v", err)
			}

			// Verify UpdateImage was called, not UpdateNote
			if client.updateImageCalls != 1 {
				t.Errorf("UpdateImage() should call UpdateImage, got %d calls", client.updateImageCalls)
			}
			if client.updateNoteCalls != 0 {
				t.Errorf("UpdateImage() should not call UpdateNote, got %d calls", client.updateNoteCalls)
			}
		})
	}
}

func TestNoteUpdater_RetryLogic(t *testing.T) {
	t.Run("succeeds on first try", func(t *testing.T) {
		client := &mockNoteClient{}
		updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 3, RetryDelay: time.Millisecond})

		err := updater.UpdateText("note-123", "test")

		if err != nil {
			t.Errorf("UpdateText() unexpected error = %v", err)
		}
		if client.updateNoteCalls != 1 {
			t.Errorf("UpdateText() call count = %d, want 1", client.updateNoteCalls)
		}
	})

	t.Run("succeeds after retry", func(t *testing.T) {
		client := &mockNoteClient{failUntil: 2} // Fail twice, then succeed
		updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 3, RetryDelay: time.Millisecond})

		err := updater.UpdateText("note-123", "test")

		if err != nil {
			t.Errorf("UpdateText() unexpected error = %v", err)
		}
		if client.updateNoteCalls != 3 {
			t.Errorf("UpdateText() call count = %d, want 3", client.updateNoteCalls)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		client := &mockNoteClient{shouldFail: true}
		updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 3, RetryDelay: time.Millisecond})

		err := updater.UpdateText("note-123", "test")

		if err == nil {
			t.Error("UpdateText() expected error, got nil")
		}
		if !errors.Is(err, ErrMaxRetriesExceeded) {
			t.Errorf("UpdateText() error should wrap ErrMaxRetriesExceeded, got %v", err)
		}
		if client.updateNoteCalls != 3 {
			t.Errorf("UpdateText() call count = %d, want 3 (max retries)", client.updateNoteCalls)
		}
	})

	t.Run("single retry on max=1", func(t *testing.T) {
		client := &mockNoteClient{shouldFail: true}
		updater := NewNoteUpdater(client, RetryConfig{MaxRetries: 1, RetryDelay: time.Millisecond})

		err := updater.UpdateText("note-123", "test")

		if err == nil {
			t.Error("UpdateText() expected error, got nil")
		}
		if client.updateNoteCalls != 1 {
			t.Errorf("UpdateText() call count = %d, want 1", client.updateNoteCalls)
		}
	})
}

func TestRetryableError(t *testing.T) {
	innerErr := errors.New("inner error")
	retryErr := &RetryableError{
		Err:         innerErr,
		Attempt:     2,
		MaxAttempts: 3,
	}

	// Test Error() method
	errStr := retryErr.Error()
	if errStr != "attempt 2/3: inner error" {
		t.Errorf("RetryableError.Error() = %q, want %q", errStr, "attempt 2/3: inner error")
	}

	// Test Unwrap() method
	unwrapped := retryErr.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("RetryableError.Unwrap() = %v, want %v", unwrapped, innerErr)
	}

	// Test errors.Is works with unwrap
	if !errors.Is(retryErr, innerErr) {
		t.Error("errors.Is should find inner error through Unwrap")
	}
}
