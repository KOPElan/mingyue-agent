package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is a simple HTTP client for making API requests to the agent
type APIClient struct {
	baseURL string
	apiKey  string
	user    string
	client  *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL, apiKey, user string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		user:    user,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Request makes an HTTP request to the API
func (c *APIClient) Request(method, path string, body interface{}) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.user != "" {
		req.Header.Set("X-User", c.user)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		return &apiResp, fmt.Errorf("API error: %s", apiResp.Error)
	}

	return &apiResp, nil
}

// Get makes a GET request
func (c *APIClient) Get(path string) (*APIResponse, error) {
	return c.Request(http.MethodGet, path, nil)
}

// Post makes a POST request
func (c *APIClient) Post(path string, body interface{}) (*APIResponse, error) {
	return c.Request(http.MethodPost, path, body)
}
