package canvasanalyzer

import (
	"testing"
)

func TestWidget_GetID(t *testing.T) {
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{
			name:   "valid ID",
			widget: Widget{"id": "widget-123"},
			want:   "widget-123",
		},
		{
			name:   "missing ID",
			widget: Widget{"type": "note"},
			want:   "",
		},
		{
			name:   "non-string ID",
			widget: Widget{"id": 123},
			want:   "",
		},
		{
			name:   "empty widget",
			widget: Widget{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.widget.GetID()
			if got != tt.want {
				t.Errorf("GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWidget_GetType(t *testing.T) {
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{
			name:   "note type",
			widget: Widget{"type": "note"},
			want:   "note",
		},
		{
			name:   "image type",
			widget: Widget{"type": "image"},
			want:   "image",
		},
		{
			name:   "missing type",
			widget: Widget{"id": "123"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.widget.GetType()
			if got != tt.want {
				t.Errorf("GetType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWidget_GetTitle(t *testing.T) {
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{
			name:   "valid title",
			widget: Widget{"title": "My Note"},
			want:   "My Note",
		},
		{
			name:   "empty title",
			widget: Widget{"title": ""},
			want:   "",
		},
		{
			name:   "missing title",
			widget: Widget{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.widget.GetTitle()
			if got != tt.want {
				t.Errorf("GetTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWidget_GetText(t *testing.T) {
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{
			name:   "valid text",
			widget: Widget{"text": "Hello world"},
			want:   "Hello world",
		},
		{
			name:   "missing text",
			widget: Widget{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.widget.GetText()
			if got != tt.want {
				t.Errorf("GetText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterWidgets(t *testing.T) {
	widgets := []Widget{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
		{"id": "4", "type": "note"},
	}

	tests := []struct {
		name       string
		excludeIDs []string
		wantCount  int
		wantIDs    []string
	}{
		{
			name:       "exclude none",
			excludeIDs: nil,
			wantCount:  4,
			wantIDs:    []string{"1", "2", "3", "4"},
		},
		{
			name:       "exclude one",
			excludeIDs: []string{"2"},
			wantCount:  3,
			wantIDs:    []string{"1", "3", "4"},
		},
		{
			name:       "exclude multiple",
			excludeIDs: []string{"1", "3"},
			wantCount:  2,
			wantIDs:    []string{"2", "4"},
		},
		{
			name:       "exclude nonexistent",
			excludeIDs: []string{"99"},
			wantCount:  4,
			wantIDs:    []string{"1", "2", "3", "4"},
		},
		{
			name:       "exclude all",
			excludeIDs: []string{"1", "2", "3", "4"},
			wantCount:  0,
			wantIDs:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterWidgets(widgets, tt.excludeIDs...)
			if len(result) != tt.wantCount {
				t.Errorf("FilterWidgets() count = %v, want %v", len(result), tt.wantCount)
			}

			// Verify IDs
			gotIDs := make([]string, len(result))
			for i, w := range result {
				gotIDs[i] = w.GetID()
			}
			for i, wantID := range tt.wantIDs {
				if i >= len(gotIDs) || gotIDs[i] != wantID {
					t.Errorf("FilterWidgets() IDs = %v, want %v", gotIDs, tt.wantIDs)
					break
				}
			}
		})
	}
}

func TestFilterWidgetsByType(t *testing.T) {
	widgets := []Widget{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "pdf"},
		{"id": "4", "type": "note"},
		{"id": "5", "type": "video"},
	}

	tests := []struct {
		name      string
		types     []string
		wantCount int
	}{
		{
			name:      "filter notes",
			types:     []string{"note"},
			wantCount: 2,
		},
		{
			name:      "filter images",
			types:     []string{"image"},
			wantCount: 1,
		},
		{
			name:      "filter multiple types",
			types:     []string{"note", "image"},
			wantCount: 3,
		},
		{
			name:      "filter nonexistent type",
			types:     []string{"browser"},
			wantCount: 0,
		},
		{
			name:      "empty filter returns all",
			types:     nil,
			wantCount: 5,
		},
		{
			name:      "case insensitive",
			types:     []string{"NOTE", "Image"},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterWidgetsByType(widgets, tt.types...)
			if len(result) != tt.wantCount {
				t.Errorf("FilterWidgetsByType() count = %v, want %v", len(result), tt.wantCount)
			}
		})
	}
}

func TestWidgetsToJSON(t *testing.T) {
	tests := []struct {
		name    string
		widgets []Widget
		wantErr bool
	}{
		{
			name: "simple widgets",
			widgets: []Widget{
				{"id": "1", "type": "note"},
				{"id": "2", "type": "image"},
			},
			wantErr: false,
		},
		{
			name:    "empty list",
			widgets: []Widget{},
			wantErr: false,
		},
		{
			name:    "nil list",
			widgets: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := WidgetsToJSON(tt.widgets)
			if (err != nil) != tt.wantErr {
				t.Errorf("WidgetsToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("WidgetsToJSON() returned empty string for valid input")
			}
		})
	}
}

func TestCountWidgetsByType(t *testing.T) {
	widgets := []Widget{
		{"id": "1", "type": "note"},
		{"id": "2", "type": "image"},
		{"id": "3", "type": "note"},
		{"id": "4", "type": "pdf"},
		{"id": "5", "type": "note"},
	}

	counts := CountWidgetsByType(widgets)

	if counts["note"] != 3 {
		t.Errorf("note count = %v, want 3", counts["note"])
	}
	if counts["image"] != 1 {
		t.Errorf("image count = %v, want 1", counts["image"])
	}
	if counts["pdf"] != 1 {
		t.Errorf("pdf count = %v, want 1", counts["pdf"])
	}
}

func TestCountWidgetsByType_Empty(t *testing.T) {
	counts := CountWidgetsByType([]Widget{})
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}

func TestSummarizeWidgets(t *testing.T) {
	tests := []struct {
		name    string
		widgets []Widget
		wantLen int // Check that result has reasonable length
	}{
		{
			name:    "empty list",
			widgets: []Widget{},
			wantLen: 10, // "no widgets"
		},
		{
			name: "single type",
			widgets: []Widget{
				{"type": "note"},
				{"type": "note"},
			},
			wantLen: 5, // "2 notes" or similar
		},
		{
			name: "multiple types",
			widgets: []Widget{
				{"type": "note"},
				{"type": "image"},
				{"type": "pdf"},
			},
			wantLen: 10, // Should have some content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeWidgets(tt.widgets)
			if len(result) < tt.wantLen {
				t.Errorf("SummarizeWidgets() = %q (len %d), want len >= %d", result, len(result), tt.wantLen)
			}
		})
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCount(tt.n)
			if got != tt.want {
				t.Errorf("formatCount(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestExtractWidgetContent(t *testing.T) {
	tests := []struct {
		name   string
		widget Widget
		want   string
	}{
		{
			name:   "title only",
			widget: Widget{"title": "My Title"},
			want:   "My Title",
		},
		{
			name:   "text only",
			widget: Widget{"text": "My Text"},
			want:   "My Text",
		},
		{
			name:   "title and text",
			widget: Widget{"title": "Title", "text": "Text"},
			want:   "Title: Text",
		},
		{
			name:   "empty widget",
			widget: Widget{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWidgetContent(tt.widget)
			if got != tt.want {
				t.Errorf("ExtractWidgetContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
