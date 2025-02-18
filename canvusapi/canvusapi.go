package canvusapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Core types and interfaces at the top
type Client struct {
	Server   string
	CanvasID string
	ApiKey   string
	HTTP     *http.Client
}

// CRITICAL NOTE:
// Widget locations are RELATIVE to their parent widget!
// To get absolute canvas coordinates:
// 1. Get the parent widget's location
// 2. Add the widget's relative location to the parent's location
// This is essential for correct widget placement

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// Core client methods
func NewClient(server, canvasID, apiKey string) *Client {
	return &Client{
		Server:   server,
		CanvasID: canvasID,
		ApiKey:   apiKey,
		HTTP:     &http.Client{},
	}
}

func NewClientFromEnv() (*Client, error) {
	server := os.Getenv("CANVUS_SERVER")
	canvasID := os.Getenv("CANVAS_ID")
	apiKey := os.Getenv("CANVUS_API_KEY")

	if server == "" || canvasID == "" || apiKey == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	return NewClient(server, canvasID, apiKey), nil
}

func (c *Client) buildURL(endpoint string) string {
	return fmt.Sprintf("%s/api/v1/canvases/%s%s",
		strings.TrimRight(c.Server, "/"),
		c.CanvasID,
		endpoint)
}

func (c *Client) Request(method, endpoint string, payload interface{}, out interface{}, subscribe bool) error {
	url := c.buildURL(endpoint)
	if subscribe {
		if strings.Contains(url, "?") {
			url += "&subscribe=true"
		} else {
			url += "?subscribe=true"
		}
	}

	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Private-Token", c.ApiKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) uploadFile(endpoint, filePath string, metadata map[string]interface{}) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add metadata as "json" part
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := writer.WriteField("json", string(metadataJSON)); err != nil {
		return nil, fmt.Errorf("failed to write json field: %w", err)
	}

	// Add file as "data" part
	part, err := writer.CreateFormFile("data", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	url := c.buildURL(endpoint)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Private-Token", c.ApiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(bodyBytes),
		}
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

// Canvas-level operations
func (c *Client) GetCanvasInfo() (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", "", nil, &response, false)
	return response, err
}

func (c *Client) Subscribe(widgetType, id string) (map[string]interface{}, error) {
	var response map[string]interface{}
	endpoint := fmt.Sprintf("/%s/%s", widgetType, id)
	err := c.Request("GET", endpoint, nil, &response, true)
	return response, err
}

// Widget methods grouped by type
// Note methods
func (c *Client) CreateNote(payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("POST", "/notes", payload, &response, false)
	return response, err
}

func (c *Client) GetNote(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/notes/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateNote(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	if _, hasColor := payload["background_color"]; hasColor {
		// Try update with current payload
		err := c.Request("PATCH", fmt.Sprintf("/notes/%s", id), payload, &response, false)
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 409 {
			// If color update failed, disable auto_text_color and retry
			disableAuto := map[string]interface{}{"auto_text_color": false}
			if err := c.Request("PATCH", fmt.Sprintf("/notes/%s", id), disableAuto, nil, false); err != nil {
				return nil, err
			}
			// Retry original update
			err = c.Request("PATCH", fmt.Sprintf("/notes/%s", id), payload, &response, false)
		}
		return response, err
	}
	err := c.Request("PATCH", fmt.Sprintf("/notes/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteNote(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/notes/%s", id), nil, nil, false)
}

// PDF methods
func (c *Client) CreatePDF(filePath string, metadata map[string]interface{}) (map[string]interface{}, error) {
	return c.uploadFile("/pdfs", filePath, metadata)
}

func (c *Client) GetPDF(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/pdfs/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdatePDF(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/pdfs/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeletePDF(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/pdfs/%s", id), nil, nil, false)
}

// Image methods
func (c *Client) CreateImage(filePath string, metadata map[string]interface{}) (map[string]interface{}, error) {
	return c.uploadFile("/images", filePath, metadata)
}

func (c *Client) GetImage(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/images/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateImage(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/images/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteImage(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/images/%s", id), nil, nil, false)
}

// Video methods
func (c *Client) CreateVideo(filePath string, metadata map[string]interface{}) (map[string]interface{}, error) {
	return c.uploadFile("/videos", filePath, metadata)
}

func (c *Client) GetVideo(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/videos/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateVideo(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/videos/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteVideo(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/videos/%s", id), nil, nil, false)
}

// Browser methods
func (c *Client) CreateBrowser(payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("POST", "/browsers", payload, &response, false)
	return response, err
}

func (c *Client) GetBrowser(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/browsers/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateBrowser(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/browsers/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteBrowser(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/browsers/%s", id), nil, nil, false)
}

func (c *Client) DownloadBrowser(id string) ([]byte, error) {
	url := c.buildURL(fmt.Sprintf("/browsers/%s/download", id))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Private-Token", c.ApiKey)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "download failed",
		}
	}

	return io.ReadAll(resp.Body)
}

// Connector methods
func (c *Client) CreateConnector(payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("POST", "/connectors", payload, &response, false)
	return response, err
}

func (c *Client) GetConnector(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/connectors/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateConnector(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/connectors/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteConnector(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/connectors/%s", id), nil, nil, false)
}

// Anchor methods
func (c *Client) CreateAnchor(payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("POST", "/anchors", payload, &response, false)
	return response, err
}

func (c *Client) GetAnchor(id string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("GET", fmt.Sprintf("/anchors/%s", id), nil, &response, subscribe)
	return response, err
}

func (c *Client) UpdateAnchor(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	var response map[string]interface{}
	err := c.Request("PATCH", fmt.Sprintf("/anchors/%s", id), payload, &response, false)
	return response, err
}

func (c *Client) DeleteAnchor(id string) error {
	return c.Request("DELETE", fmt.Sprintf("/anchors/%s", id), nil, nil, false)
}

// GetWidgets gets all widgets in the canvas
func (c *Client) GetWidgets(subscribe bool) ([]map[string]interface{}, error) {
	var response []map[string]interface{}
	url := fmt.Sprintf("/widgets")
	if subscribe {
		url += "?subscribe=true"
	}
	err := c.Request("GET", url, nil, &response, false)
	return response, err
}

// GetWidget gets a single widget by ID
func (c *Client) GetWidget(widgetID string, subscribe bool) (map[string]interface{}, error) {
	var response map[string]interface{}
	url := fmt.Sprintf("/widgets/%s", widgetID)
	if subscribe {
		url += "?subscribe=true"
	}
	err := c.Request("GET", url, nil, &response, false)
	return response, err
}

// DownloadPDF downloads a PDF file
func (c *Client) DownloadPDF(pdfID string, outputPath string) error {
	return c.downloadFile(fmt.Sprintf("/pdfs/%s", pdfID), outputPath)
}

// DownloadImage downloads an image file
func (c *Client) DownloadImage(imageID string, localPath string) error {
	return c.downloadFile(fmt.Sprintf("/images/%s", imageID), localPath)
}

// DownloadVideo downloads a video file
func (c *Client) DownloadVideo(videoID string, outputPath string) error {
	return c.downloadFile(fmt.Sprintf("/videos/%s", videoID), outputPath)
}

// downloadFile is a helper function to download files
func (c *Client) downloadFile(endpoint string, outputPath string) error {
	url := fmt.Sprintf("%s/api/v1/canvases/%s%s/download",
		c.Server,
		c.CanvasID,
		endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Private-Token", c.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// SubscribeToWidgets creates a subscription to the widgets stream
func (c *Client) SubscribeToWidgets(ctx context.Context) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/api/v1/canvases/%s/widgets?subscribe", c.Server, c.CanvasID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Private-Token", c.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stream: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *Client) DeleteWidget(widgetID string) error {
	endpoint := fmt.Sprintf("/widgets/%s", widgetID)
	return c.Request("DELETE", endpoint, nil, nil, false)
}
