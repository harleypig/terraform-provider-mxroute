package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// MXroute API error codes — the "code" field of a failed envelope.
const (
	errCodeNotFound    = "NOT_FOUND"
	errCodeConflict    = "CONFLICT"
	errCodeRateLimited = "RATE_LIMITED"
	errCodeServer      = "SERVER_ERROR"
)

// Rate-limit retry policy applied to RATE_LIMITED responses.
const (
	// maxRateLimitRetries bounds how many times a rate-limited request is
	// retried before the RATE_LIMITED error is returned.
	maxRateLimitRetries = 3

	// defaultRetryDelay is used when a 429 carries no usable rate-limit hint.
	defaultRetryDelay = 1 * time.Second

	// maxRetryWait caps how long a single retry will wait.
	maxRetryWait = 60 * time.Second
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
	// RetryAfter is the delay before retrying a RATE_LIMITED request, taken
	// from the Retry-After / X-RateLimit-Reset headers (or a default backoff
	// when the response gives no hint). Zero for non-rate-limit errors.
	RetryAfter time.Duration
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
	return apiErrorHasCode(err, errCodeNotFound)
}

// IsConflict reports whether err is an *APIError with a CONFLICT code — the
// resource already exists (e.g. a create against an existing name).
func IsConflict(err error) bool {
	return apiErrorHasCode(err, errCodeConflict)
}

// IsRateLimited reports whether err is an *APIError with a RATE_LIMITED code.
// The client already retries these; a surfaced RATE_LIMITED means the retries
// were exhausted.
func IsRateLimited(err error) bool {
	return apiErrorHasCode(err, errCodeRateLimited)
}

// apiErrorHasCode reports whether err is an *APIError carrying code.
func apiErrorHasCode(err error, code string) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.Code == code
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
//
// A RATE_LIMITED (429) response is retried up to maxRateLimitRetries times,
// waiting the interval the response advertises (Retry-After /
// X-RateLimit-Reset) before each retry. A 429 rejects the request before it
// takes effect, so retrying is safe for every method.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var bodyBytes []byte

	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		bodyBytes = encoded
	}

	for attempt := 0; ; attempt++ {
		err := c.doOnce(ctx, method, path, bodyBytes, out)

		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Code != errCodeRateLimited || attempt >= maxRateLimitRetries {
			return err
		}

		wait := apiErr.RetryAfter
		if wait > maxRetryWait {
			wait = maxRetryWait
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(wait):
		}
	}
}

// doOnce performs a single request/response cycle for Do.
func (c *Client) doOnce(ctx context.Context, method, path string, bodyBytes []byte, out any) error {
	var reqBody io.Reader

	if bodyBytes != nil {
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("X-Server", c.server)
	req.Header.Set("X-Username", c.username)
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	if bodyBytes != nil {
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
			Code:       errCodeServer,
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
			Code:       errCodeServer,
			Message:    http.StatusText(resp.StatusCode),
		}

		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
			apiErr.Field = env.Error.Field
		}

		if apiErr.Code == errCodeRateLimited {
			apiErr.RetryAfter = retryAfterFromHeaders(resp.Header)
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

// retryAfterFromHeaders derives the wait before retrying a rate-limited
// request from the response headers, preferring Retry-After (seconds) then
// X-RateLimit-Reset (a Unix timestamp). It returns defaultRetryDelay when the
// response carries no usable hint.
func retryAfterFromHeaders(h http.Header) time.Duration {
	if ra := h.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
	}

	if reset := h.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			if d := time.Until(time.Unix(ts, 0)); d > 0 {
				return d
			}

			return 0
		}
	}

	return defaultRetryDelay
}
