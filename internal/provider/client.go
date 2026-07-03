package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// defaultBaseURL is the MXroute REST API root; ClientConfig.BaseURL
	// overrides it (e.g. for tests against an httptest server).
	defaultBaseURL = "https://api.mxroute.com"

	// defaultTimeout bounds a single API request.
	defaultTimeout = 30 * time.Second
)

// Client is a thin REST client for the MXroute API. It sets the three
// authentication headers on every request, unwraps the
// {success, data, error} response envelope, and maps a success:false
// envelope to an *APIError. Resources marshal their own request and
// response types and call Do, so adding a resource never edits this file.
type Client struct {
	baseURL    string
	server     string
	username   string
	apiKey     string
	httpClient *http.Client
}

// ClientConfig holds the values needed to reach and authenticate with the
// MXroute API.
type ClientConfig struct {
	// BaseURL overrides the default API root; leave empty for production.
	BaseURL string
	// Server is the mail server hostname sent as X-Server.
	Server string
	// Username is the DirectAdmin username sent as X-Username.
	Username string
	// APIKey is the API key sent as X-API-Key.
	APIKey string
	// HTTPClient overrides the default HTTP client (mainly for tests).
	HTTPClient *http.Client
}

// NewClient builds a Client from cfg, applying defaults for the base URL,
// timeout, and HTTP client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		server:     cfg.Server,
		username:   cfg.Username,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}
}

// APIError is a structured MXroute API error — the "error" object of a
// success:false envelope, carrying the HTTP status for context.
type APIError struct {
	// StatusCode is the HTTP status the error arrived with.
	StatusCode int
	// Code is the API error code, e.g. VALIDATION_ERROR or NOT_FOUND.
	Code string
	// Message is the human-readable error message.
	Message string
	// Field names the offending input on a validation error; empty otherwise.
	Field string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("mxroute API %s (HTTP %d): %s [field: %s]", e.Code, e.StatusCode, e.Message, e.Field)
	}

	return fmt.Sprintf("mxroute API %s (HTTP %d): %s", e.Code, e.StatusCode, e.Message)
}

// IsNotFound reports whether err is an *APIError with a NOT_FOUND code —
// the signal a resource's Read uses to drop the resource from state.
func IsNotFound(err error) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.Code == "NOT_FOUND"
}

// envelope is the {success, data, error} wrapper every response carries.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *envelopeError  `json:"error"`
}

// envelopeError is the "error" object of a failed envelope.
type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

// Do executes an authenticated request against path (for example
// "/domains" or "/domains/example.com/email-accounts"). When body is
// non-nil it is JSON-encoded as the request body; when out is non-nil the
// envelope's data field is unmarshaled into it. A success:false envelope,
// or a non-2xx status with no envelope, is returned as an *APIError;
// transport and decoding problems are returned as wrapped errors.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader

	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("X-Server", c.server)
	req.Header.Set("X-Username", c.username)
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// A successful DELETE or PATCH may return 204 with an empty body and no
	// envelope to decode.
	if len(bytes.TrimSpace(raw)) == 0 {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return nil
		}

		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       "SERVER_ERROR",
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	var env envelope

	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("decoding response envelope (%s %s, HTTP %d): %w", method, path, resp.StatusCode, err)
	}

	if !env.Success {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			Code:       "SERVER_ERROR",
			Message:    http.StatusText(resp.StatusCode),
		}

		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
			apiErr.Field = env.Error.Field
		}

		return apiErr
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decoding response data (%s %s): %w", method, path, err)
		}
	}

	return nil
}
