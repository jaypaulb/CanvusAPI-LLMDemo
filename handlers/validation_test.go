package handlers_test

import (
	"errors"
	"testing"

	"go_backend/handlers"
)

func TestValidateUpdate(t *testing.T) {
	tests := []struct {
		name    string
		update  handlers.Update
		wantErr error
	}{
		{
			name: "valid update with all required fields",
			update: handlers.Update{
				"id":          "test-id-123",
				"widget_type": "Note",
				"location":    map[string]interface{}{"x": 100.0, "y": 200.0},
				"size":        map[string]interface{}{"width": 400.0, "height": 300.0},
			},
			wantErr: nil,
		},
		{
			name: "valid update with extra fields",
			update: handlers.Update{
				"id":          "test-id-456",
				"widget_type": "Image",
				"location":    map[string]interface{}{"x": 0.0, "y": 0.0},
				"size":        map[string]interface{}{"width": 100.0, "height": 100.0},
				"text":        "Some text content",
				"parent_id":   "parent-123",
			},
			wantErr: nil,
		},
		{
			name:    "missing id",
			update:  handlers.Update{"widget_type": "Note", "location": map[string]interface{}{}, "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingID,
		},
		{
			name:    "empty id",
			update:  handlers.Update{"id": "", "widget_type": "Note", "location": map[string]interface{}{}, "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingID,
		},
		{
			name:    "id wrong type",
			update:  handlers.Update{"id": 123, "widget_type": "Note", "location": map[string]interface{}{}, "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingID,
		},
		{
			name:    "missing widget_type",
			update:  handlers.Update{"id": "test-id", "location": map[string]interface{}{}, "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingType,
		},
		{
			name:    "empty widget_type",
			update:  handlers.Update{"id": "test-id", "widget_type": "", "location": map[string]interface{}{}, "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingType,
		},
		{
			name:    "missing location",
			update:  handlers.Update{"id": "test-id", "widget_type": "Note", "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingLocation,
		},
		{
			name:    "location wrong type",
			update:  handlers.Update{"id": "test-id", "widget_type": "Note", "location": "invalid", "size": map[string]interface{}{}},
			wantErr: handlers.ErrMissingLocation,
		},
		{
			name:    "missing size",
			update:  handlers.Update{"id": "test-id", "widget_type": "Note", "location": map[string]interface{}{}},
			wantErr: handlers.ErrMissingSize,
		},
		{
			name:    "size wrong type",
			update:  handlers.Update{"id": "test-id", "widget_type": "Note", "location": map[string]interface{}{}, "size": []int{100, 200}},
			wantErr: handlers.ErrMissingSize,
		},
		{
			name:    "empty update",
			update:  handlers.Update{},
			wantErr: handlers.ErrMissingID,
		},
		{
			name:    "nil update",
			update:  nil,
			wantErr: handlers.ErrMissingID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.ValidateUpdate(tt.update)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateUpdate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateUpdate() expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestValidateUpdateField(t *testing.T) {
	update := handlers.Update{
		"string_field": "hello",
		"int_field":    42,
		"float_field":  3.14,
		"map_field":    map[string]interface{}{"nested": "value"},
		"nil_field":    nil,
	}

	t.Run("valid string field", func(t *testing.T) {
		val, err := handlers.ValidateUpdateField[string](update, "string_field")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != "hello" {
			t.Errorf("got %q, want %q", val, "hello")
		}
	})

	t.Run("valid int field", func(t *testing.T) {
		val, err := handlers.ValidateUpdateField[int](update, "int_field")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val != 42 {
			t.Errorf("got %d, want %d", val, 42)
		}
	})

	t.Run("valid map field", func(t *testing.T) {
		val, err := handlers.ValidateUpdateField[map[string]interface{}](update, "map_field")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if val["nested"] != "value" {
			t.Errorf("got %v, want nested=value", val)
		}
	})

	t.Run("missing field", func(t *testing.T) {
		_, err := handlers.ValidateUpdateField[string](update, "nonexistent")
		if err == nil {
			t.Error("expected error for missing field")
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		_, err := handlers.ValidateUpdateField[string](update, "int_field")
		if err == nil {
			t.Error("expected error for wrong type")
		}
	})
}

func TestValidateNonEmptyString(t *testing.T) {
	tests := []struct {
		name      string
		update    handlers.Update
		fieldName string
		want      string
		wantErr   bool
	}{
		{
			name:      "valid non-empty string",
			update:    handlers.Update{"text": "hello world"},
			fieldName: "text",
			want:      "hello world",
			wantErr:   false,
		},
		{
			name:      "empty string",
			update:    handlers.Update{"text": ""},
			fieldName: "text",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "missing field",
			update:    handlers.Update{},
			fieldName: "text",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "wrong type",
			update:    handlers.Update{"text": 123},
			fieldName: "text",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handlers.ValidateNonEmptyString(tt.update, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNonEmptyString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ValidateNonEmptyString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasField(t *testing.T) {
	update := handlers.Update{
		"exists":    "value",
		"nil_value": nil,
		"empty":     "",
	}

	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		{"existing field", "exists", true},
		{"nil value field", "nil_value", true},
		{"empty string field", "empty", true},
		{"non-existent field", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.HasField(update, tt.fieldName)
			if got != tt.want {
				t.Errorf("HasField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringField(t *testing.T) {
	update := handlers.Update{
		"text":     "hello",
		"empty":    "",
		"number":   42,
		"nilvalue": nil,
	}

	tests := []struct {
		name         string
		fieldName    string
		defaultValue string
		want         string
	}{
		{"existing string", "text", "default", "hello"},
		{"empty string", "empty", "default", ""},
		{"wrong type returns default", "number", "default", "default"},
		{"missing field returns default", "missing", "default", "default"},
		{"nil value returns default", "nilvalue", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handlers.GetStringField(update, tt.fieldName, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetStringField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetMapField(t *testing.T) {
	validMap := map[string]interface{}{"x": 1.0, "y": 2.0}
	update := handlers.Update{
		"location": validMap,
		"string":   "not a map",
		"nilvalue": nil,
	}

	t.Run("existing map", func(t *testing.T) {
		got := handlers.GetMapField(update, "location")
		if got == nil {
			t.Error("expected non-nil map")
		}
		if got["x"] != 1.0 {
			t.Errorf("got x=%v, want 1.0", got["x"])
		}
	})

	t.Run("wrong type returns nil", func(t *testing.T) {
		got := handlers.GetMapField(update, "string")
		if got != nil {
			t.Errorf("expected nil for wrong type, got %v", got)
		}
	})

	t.Run("missing field returns nil", func(t *testing.T) {
		got := handlers.GetMapField(update, "missing")
		if got != nil {
			t.Errorf("expected nil for missing field, got %v", got)
		}
	})

	t.Run("nil value returns nil", func(t *testing.T) {
		got := handlers.GetMapField(update, "nilvalue")
		if got != nil {
			t.Errorf("expected nil for nil value, got %v", got)
		}
	})
}
