// Package ollama provides a client for interacting with the Ollama API.
// It supports listing local and running models, with utilities for formatting and timeouts.
package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client represents an HTTP client for communicating with an Ollama server.
// It provides methods to query model information and manage model operations.
type Client struct {
	BaseURL string       // Base URL of the Ollama server (e.g., "http://localhost:11434")
	HTTP    *http.Client // HTTP client for making requests
}

// NewClient creates a new Ollama client with the specified base URL.
// If base is empty, it defaults to "http://localhost:11434".
// The HTTP client is configured with no timeout for long-running operations.
func NewClient(base string) *Client {
	if base == "" {
		base = "http://localhost:11434"
	}
	return &Client{
		BaseURL: base,
		HTTP:    &http.Client{Timeout: 0},
	}
}

// Model represents an Ollama model with its metadata.
// It contains the model name, optional digest for identification, and size in bytes.
type Model struct {
	Name   string `json:"name"`             // Model name (e.g., "llama2:7b")
	Digest string `json:"digest,omitempty"` // SHA256 digest of the model
	Size   int64  `json:"size,omitempty"`   // Model size in bytes
}

// ListLocalModels retrieves all locally installed models from the Ollama server.
// It makes a GET request to /api/tags and returns the list of available models.
func (c *Client) ListLocalModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tags: %s", res.Status)
	}
	var payload struct {
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

// ListRunning retrieves all currently running models from the Ollama server.
// It makes a GET request to /api/ps and returns the list of active models.
func (c *Client) ListRunning(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/ps", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ps: %s", res.Status)
	}
	var payload struct {
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

// HumanSize formats a byte count into a human-readable string.
// It converts bytes to KiB, MiB, or GiB as appropriate, or returns "-" for zero/negative values.
func HumanSize(n int64) string {
	const (
		kib = 1024
		mib = kib * 1024
		gib = mib * 1024
	)
	switch {
	case n >= gib:
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(gib))
	case n >= mib:
		return fmt.Sprintf("%.2f MiB", float64(n)/float64(mib))
	case n >= kib:
		return fmt.Sprintf("%.2f KiB", float64(n)/float64(kib))
	case n > 0:
		return fmt.Sprintf("%d B", n)
	default:
		return "-"
	}
}

// WithTimeout creates a context with timeout if the duration is positive.
// If duration is zero or negative, it returns the original context and a no-op cancel function.
func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
