package tests

// TestData holds test payloads
type TestData struct {
	CreatePayload map[string]interface{}
	UpdatePayload map[string]interface{}
}

// FileTestData extends TestData with file-specific information
type FileTestData struct {
	TestData
	FilePath string
}

// GetNoteTestData returns test data for note widget operations
func GetNoteTestData() TestData {
	return TestData{
		CreatePayload: map[string]interface{}{
			"title":           "Test Note",
			"text":            "This is a test note",
			"depth":           0,
			"pinned":          false,
			"auto_text_color": true,
			"location": map[string]interface{}{
				"x": 1000.0,
				"y": 1000.0,
			},
			"size": map[string]interface{}{
				"width":  200.0,
				"height": 150.0,
			},
			"state":       "normal",
			"widget_type": "Note",
			"scale":       1.0,
		},
		UpdatePayload: map[string]interface{}{
			"title":            "Updated Note",
			"text":             "This note has been updated",
			"auto_text_color":  false,
			"background_color": "#FF0000FF",
			"text_color":       "#FFFFFFFF",
		},
	}
}

// GetPDFTestData returns test data for PDF widget operations
func GetPDFTestData() FileTestData {
	return FileTestData{
		TestData: TestData{
			CreatePayload: map[string]interface{}{
				"title":  "Test PDF",
				"depth":  0,
				"pinned": false,
				"scale":  1.0,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
				"size": map[string]interface{}{
					"width":  595.0,
					"height": 842.0,
				},
				"state":       "normal",
				"widget_type": "Pdf",
			},
			UpdatePayload: map[string]interface{}{
				"title": "Updated PDF",
				"scale": 1.5,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
			},
		},
		FilePath: "../test_files/test_pdf.pdf",
	}
}

// GetImageTestData returns test data for image widget operations
func GetImageTestData() FileTestData {
	return FileTestData{
		TestData: TestData{
			CreatePayload: map[string]interface{}{
				"title":  "Test Image",
				"depth":  0,
				"pinned": false,
				"scale":  1.0,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
				"size": map[string]interface{}{
					"width":  800.0,
					"height": 600.0,
				},
				"state":       "normal",
				"widget_type": "Image",
			},
			UpdatePayload: map[string]interface{}{
				"title": "Updated Image",
				"scale": 2.0,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
			},
		},
		FilePath: "../test_files/test_image.jpg",
	}
}

// GetVideoTestData returns test data for video widget operations
func GetVideoTestData() FileTestData {
	return FileTestData{
		TestData: TestData{
			CreatePayload: map[string]interface{}{
				"title":             "Test Video",
				"depth":             0,
				"pinned":            false,
				"playback_position": 0,
				"playback_state":    "STOPPED",
				"scale":             1.0,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
				"size": map[string]interface{}{
					"width":  640.0,
					"height": 360.0,
				},
				"state":       "normal",
				"widget_type": "Video",
			},
			UpdatePayload: map[string]interface{}{
				"title":          "Updated Video",
				"playback_state": "PLAYING",
				"scale":          1.5,
				"location": map[string]interface{}{
					"x": 1000.0,
					"y": 1000.0,
				},
			},
		},
		FilePath: "../test_files/test_video.mp4",
	}
}

// GetBrowserTestData returns test data for browser widget operations
func GetBrowserTestData() TestData {
	return TestData{
		CreatePayload: map[string]interface{}{
			"url":    "https://example.com",
			"depth":  0,
			"pinned": false,
			"location": map[string]interface{}{
				"x": 1000.0,
				"y": 1000.0,
			},
			"size": map[string]interface{}{
				"width":  1024.0,
				"height": 768.0,
			},
			"state":       "normal",
			"widget_type": "Browser",
			"scale":       1.0,
		},
		UpdatePayload: map[string]interface{}{
			"url": "https://updated-example.com",
		},
	}
}

// GetConnectorTestData returns test data for connector widget operations
func GetConnectorTestData() TestData {
	return TestData{
		CreatePayload: map[string]interface{}{
			"widget_type": "Connector",
			"type":        "curve",
			"state":       "normal",
			"line_color":  "#FF0000FF",
			"line_width":  2.0,
			"src": map[string]interface{}{
				"id":            "", // Will be filled with source note ID
				"auto_location": false,
				"rel_location": map[string]interface{}{
					"x": 1.0, // Right edge of source
					"y": 0.5, // Vertically centered
				},
				"tip": "none",
			},
			"dst": map[string]interface{}{
				"id":            "", // Will be filled with destination note ID
				"auto_location": true,
				"rel_location": map[string]interface{}{
					"x": 0.0, // Left edge of destination
					"y": 0.5, // Vertically centered
				},
				"tip": "solid-equilateral-triangle",
			},
		},
		UpdatePayload: map[string]interface{}{
			"line_color": "#00FF00FF",
			"line_width": 4.0,
		},
	}
}

// GetAnchorTestData returns test data for anchor widget operations
func GetAnchorTestData() TestData {
	return TestData{
		CreatePayload: map[string]interface{}{
			"anchor_name": "Test Anchor",
			"depth":       0,
			"pinned":      false,
			"location": map[string]interface{}{
				"x": 1000.0,
				"y": 1000.0,
			},
			"widget_type": "Anchor",
			"scale":       1.0,
		},
		UpdatePayload: map[string]interface{}{
			"anchor_name": "Updated Anchor",
		},
	}
}
