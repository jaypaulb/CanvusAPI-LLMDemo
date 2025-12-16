package handlers

import (
	"errors"
	"testing"
)

func TestExtractJSONFromText(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr error
	}{
		{
			name:    "simple JSON",
			text:    `{"key": "value"}`,
			want:    `{"key": "value"}`,
			wantErr: nil,
		},
		{
			name:    "JSON with surrounding text",
			text:    `Some text before {"type": "text", "content": "hello"} and after`,
			want:    `{"type": "text", "content": "hello"}`,
			wantErr: nil,
		},
		{
			name:    "JSON with leading text only",
			text:    `Here is the response: {"type": "image"}`,
			want:    `{"type": "image"}`,
			wantErr: nil,
		},
		{
			name:    "JSON with trailing text only",
			text:    `{"content": "test"} - that's it`,
			want:    `{"content": "test"}`,
			wantErr: nil,
		},
		{
			name:    "nested JSON objects",
			text:    `{"outer": {"inner": "value"}}`,
			want:    `{"outer": {"inner": "value"}}`,
			wantErr: nil,
		},
		{
			name:    "no JSON - empty string",
			text:    "",
			want:    "",
			wantErr: ErrNoJSONFound,
		},
		{
			name:    "no JSON - plain text",
			text:    "This is just plain text with no JSON",
			want:    "",
			wantErr: ErrNoJSONFound,
		},
		{
			name:    "only opening brace",
			text:    "Missing closing { brace",
			want:    "",
			wantErr: ErrNoJSONFound,
		},
		{
			name:    "only closing brace",
			text:    "Missing opening } brace",
			want:    "",
			wantErr: ErrNoJSONFound,
		},
		{
			name:    "braces in wrong order",
			text:    "} before {",
			want:    "",
			wantErr: ErrNoJSONFound,
		},
		{
			name:    "multiline JSON",
			text:    "Response:\n{\n  \"type\": \"text\",\n  \"content\": \"hello\"\n}\nEnd",
			want:    "{\n  \"type\": \"text\",\n  \"content\": \"hello\"\n}",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractJSONFromText(tt.text)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ExtractJSONFromText() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ExtractJSONFromText() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractJSONFromText() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ExtractJSONFromText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseJSONToMap(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		wantKeys  []string
		wantErr   bool
		checkVals map[string]interface{}
	}{
		{
			name:     "simple object",
			jsonStr:  `{"key": "value"}`,
			wantKeys: []string{"key"},
			wantErr:  false,
			checkVals: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name:     "multiple fields",
			jsonStr:  `{"type": "text", "content": "hello"}`,
			wantKeys: []string{"type", "content"},
			wantErr:  false,
			checkVals: map[string]interface{}{
				"type":    "text",
				"content": "hello",
			},
		},
		{
			name:     "number value",
			jsonStr:  `{"count": 42}`,
			wantKeys: []string{"count"},
			wantErr:  false,
			checkVals: map[string]interface{}{
				"count": float64(42),
			},
		},
		{
			name:     "boolean value",
			jsonStr:  `{"active": true}`,
			wantKeys: []string{"active"},
			wantErr:  false,
			checkVals: map[string]interface{}{
				"active": true,
			},
		},
		{
			name:     "empty object",
			jsonStr:  `{}`,
			wantKeys: []string{},
			wantErr:  false,
		},
		{
			name:    "invalid JSON - missing quote",
			jsonStr: `{key: "value"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON - trailing comma",
			jsonStr: `{"key": "value",}`,
			wantErr: true,
		},
		{
			name:    "not JSON at all",
			jsonStr: "just plain text",
			wantErr: true,
		},
		{
			name:    "empty string",
			jsonStr: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJSONToMap(tt.jsonStr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseJSONToMap() expected error, got nil")
				}
				if !errors.Is(err, ErrInvalidJSON) {
					t.Errorf("ParseJSONToMap() error should wrap ErrInvalidJSON, got %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseJSONToMap() unexpected error = %v", err)
				return
			}

			// Check expected keys exist
			for _, key := range tt.wantKeys {
				if _, exists := got[key]; !exists {
					t.Errorf("ParseJSONToMap() missing expected key %q", key)
				}
			}

			// Check specific values
			for key, want := range tt.checkVals {
				if got[key] != want {
					t.Errorf("ParseJSONToMap()[%q] = %v, want %v", key, got[key], want)
				}
			}
		})
	}
}

func TestValidateAIResponseFields(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr error
	}{
		{
			name: "valid response",
			data: map[string]interface{}{
				"type":    "text",
				"content": "hello world",
			},
			wantErr: nil,
		},
		{
			name: "valid image response",
			data: map[string]interface{}{
				"type":    "image",
				"content": "a beautiful sunset",
			},
			wantErr: nil,
		},
		{
			name: "missing type",
			data: map[string]interface{}{
				"content": "hello",
			},
			wantErr: ErrMissingTypeField,
		},
		{
			name: "missing content",
			data: map[string]interface{}{
				"type": "text",
			},
			wantErr: ErrMissingContentField,
		},
		{
			name:    "empty map",
			data:    map[string]interface{}{},
			wantErr: ErrMissingTypeField,
		},
		{
			name: "type is not string",
			data: map[string]interface{}{
				"type":    123,
				"content": "hello",
			},
			wantErr: ErrMissingTypeField,
		},
		{
			name: "content is not string",
			data: map[string]interface{}{
				"type":    "text",
				"content": 123,
			},
			wantErr: ErrMissingContentField,
		},
		{
			name: "extra fields are allowed",
			data: map[string]interface{}{
				"type":    "text",
				"content": "hello",
				"extra":   "ignored",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAIResponseFields(tt.data)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateAIResponseFields() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateAIResponseFields() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateAIResponseFields() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateContentField(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]interface{}
		wantErr error
	}{
		{
			name: "valid content only",
			data: map[string]interface{}{
				"content": "hello world",
			},
			wantErr: nil,
		},
		{
			name: "with type field",
			data: map[string]interface{}{
				"type":    "text",
				"content": "hello",
			},
			wantErr: nil,
		},
		{
			name:    "missing content",
			data:    map[string]interface{}{},
			wantErr: ErrMissingContentField,
		},
		{
			name: "content is not string",
			data: map[string]interface{}{
				"content": 123,
			},
			wantErr: ErrMissingContentField,
		},
		{
			name: "content is nil",
			data: map[string]interface{}{
				"content": nil,
			},
			wantErr: ErrMissingContentField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContentField(tt.data)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ValidateContentField() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateContentField() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateContentField() unexpected error = %v", err)
			}
		})
	}
}

func TestExtractAndParseAIResponse(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantType    string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "valid text response",
			text:        `{"type": "text", "content": "Hello world"}`,
			wantType:    "text",
			wantContent: "Hello world",
			wantErr:     false,
		},
		{
			name:        "valid image response",
			text:        `{"type": "image", "content": "a sunset over mountains"}`,
			wantType:    "image",
			wantContent: "a sunset over mountains",
			wantErr:     false,
		},
		{
			name:        "response with surrounding text",
			text:        `Here is my answer: {"type": "text", "content": "42"} Hope that helps!`,
			wantType:    "text",
			wantContent: "42",
			wantErr:     false,
		},
		{
			name:    "no JSON in text",
			text:    "Just plain text without JSON",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			text:    `{type: text}`,
			wantErr: true,
		},
		{
			name:    "missing type field",
			text:    `{"content": "hello"}`,
			wantErr: true,
		},
		{
			name:    "missing content field",
			text:    `{"type": "text"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ExtractAndParseAIResponse(tt.text)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractAndParseAIResponse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractAndParseAIResponse() unexpected error = %v", err)
				return
			}

			if resp.Type != tt.wantType {
				t.Errorf("ExtractAndParseAIResponse().Type = %v, want %v", resp.Type, tt.wantType)
			}
			if resp.Content != tt.wantContent {
				t.Errorf("ExtractAndParseAIResponse().Content = %v, want %v", resp.Content, tt.wantContent)
			}
		})
	}
}

func TestExtractAndParseContent(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name:    "valid content",
			text:    `{"content": "hello world"}`,
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "content with type",
			text:    `{"type": "text", "content": "with type"}`,
			want:    "with type",
			wantErr: false,
		},
		{
			name:    "with surrounding text",
			text:    `Response: {"content": "extracted"} done`,
			want:    "extracted",
			wantErr: false,
		},
		{
			name:    "no JSON",
			text:    "no json here",
			wantErr: true,
		},
		{
			name:    "missing content field",
			text:    `{"type": "text"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractAndParseContent(tt.text)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractAndParseContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExtractAndParseContent() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ExtractAndParseContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringFieldFromJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		fieldName string
		want      string
		wantErr   bool
	}{
		{
			name:      "existing field",
			jsonStr:   `{"name": "test"}`,
			fieldName: "name",
			want:      "test",
			wantErr:   false,
		},
		{
			name:      "content field",
			jsonStr:   `{"type": "text", "content": "hello"}`,
			fieldName: "content",
			want:      "hello",
			wantErr:   false,
		},
		{
			name:      "missing field",
			jsonStr:   `{"other": "value"}`,
			fieldName: "name",
			wantErr:   true,
		},
		{
			name:      "field is not string",
			jsonStr:   `{"count": 42}`,
			fieldName: "count",
			wantErr:   true,
		},
		{
			name:      "invalid JSON",
			jsonStr:   "not json",
			fieldName: "any",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStringFieldFromJSON(tt.jsonStr, tt.fieldName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetStringFieldFromJSON() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetStringFieldFromJSON() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("GetStringFieldFromJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeNewlines(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single escaped newline",
			text: "Hello\\nWorld",
			want: "Hello\nWorld",
		},
		{
			name: "multiple escaped newlines",
			text: "Line1\\nLine2\\nLine3",
			want: "Line1\nLine2\nLine3",
		},
		{
			name: "no escaped newlines",
			text: "Just plain text",
			want: "Just plain text",
		},
		{
			name: "empty string",
			text: "",
			want: "",
		},
		{
			name: "actual newlines preserved",
			text: "Line1\nLine2\\nLine3",
			want: "Line1\nLine2\nLine3",
		},
		{
			name: "consecutive escaped newlines",
			text: "Para1\\n\\nPara2",
			want: "Para1\n\nPara2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeNewlines(tt.text)
			if got != tt.want {
				t.Errorf("NormalizeNewlines() = %q, want %q", got, tt.want)
			}
		})
	}
}
