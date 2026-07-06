package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

// newTestClient returns a Client pointed at srv with canned credentials.
func newTestClient(srv *httptest.Server) *Client {
	return NewClient(ClientConfig{
		BaseURL:    srv.URL,
		Server:     "heracles.mxrouting.net",
		Username:   "harleypi",
		APIKey:     "test-key",
		HTTPClient: srv.Client(),
	})
}

func TestDoSetsAuthHeadersAndDecodesData(t *testing.T) {
	var gotHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":["example.com","harleypig.com"]}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	var domains []string
	if err := client.Do(t.Context(), http.MethodGet, "/domains", nil, &domains); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}

	want := []string{"example.com", "harleypig.com"}
	if !slices.Equal(domains, want) {
		t.Errorf("domains = %v, want %v", domains, want)
	}

	for header, wantValue := range map[string]string{
		"X-Server":   "heracles.mxrouting.net",
		"X-Username": "harleypi",
		"X-API-Key":  "test-key",
	} {
		if got := gotHeaders.Get(header); got != wantValue {
			t.Errorf("header %s = %q, want %q", header, got, wantValue)
		}
	}
}

func TestDoSendsJSONBody(t *testing.T) {
	var (
		gotBody        map[string]any
		gotContentType string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"success":true,"data":{"domain":"example.com","ssl_enabled":true}}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	var out struct {
		Domain     string `json:"domain"`
		SSLEnabled bool   `json:"ssl_enabled"`
	}

	body := map[string]string{"domain": "example.com"}
	if err := client.Do(t.Context(), http.MethodPost, "/domains", body, &out); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	if gotBody["domain"] != "example.com" {
		t.Errorf("request body domain = %v, want example.com", gotBody["domain"])
	}

	if out.Domain != "example.com" || !out.SSLEnabled {
		t.Errorf("decoded out = %+v, want {example.com true}", out)
	}
}

func TestDoMapsErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"VALIDATION_ERROR","message":"Invalid domain format","field":"domain"}}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	err := client.Do(t.Context(), http.MethodPost, "/domains", map[string]string{"domain": "bad"}, nil)

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v (%T), want *APIError", err, err)
	}

	if apiErr.Code != "VALIDATION_ERROR" || apiErr.Field != "domain" || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"Domain not found"}}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	err := client.Do(t.Context(), http.MethodGet, "/domains/missing.com", nil, nil)
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound(%v) = false, want true", err)
	}

	if IsNotFound(nil) {
		t.Error("IsNotFound(nil) = true, want false")
	}

	if IsNotFound(errors.New("boom")) {
		t.Error("IsNotFound(non-API error) = true, want false")
	}
}

func TestDoEmptyBodySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(srv)

	if err := client.Do(t.Context(), http.MethodDelete, "/domains/example.com", nil, nil); err != nil {
		t.Fatalf("Do returned error on 204: %v", err)
	}
}

func TestDoMalformedEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	err := client.Do(t.Context(), http.MethodGet, "/domains", nil, nil)
	if err == nil {
		t.Fatal("Do returned nil, want decode error")
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("error mapped to *APIError, want raw decode error: %v", err)
	}
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient(ClientConfig{Server: "s", Username: "u", APIKey: "k"})
	if client.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil, want default")
	}

	trimmed := NewClient(ClientConfig{BaseURL: "https://example.test/"})
	if trimmed.baseURL != "https://example.test" {
		t.Errorf("baseURL = %q, want trailing slash trimmed", trimmed.baseURL)
	}
}

func TestDoMapsAllErrorCodes(t *testing.T) {
	cases := []struct {
		code   string
		status int
	}{
		{"VALIDATION_ERROR", http.StatusBadRequest},
		{"UNAUTHORIZED", http.StatusUnauthorized},
		{"FORBIDDEN", http.StatusForbidden},
		{"NOT_FOUND", http.StatusNotFound},
		{"CONFLICT", http.StatusConflict},
		{"BUSINESS_ERROR", http.StatusUnprocessableEntity},
		{"RATE_LIMITED", http.StatusTooManyRequests},
		{"SERVER_ERROR", http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Retry-After 0 keeps the RATE_LIMITED retries instant.
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"success":false,"error":{"code":%q,"message":"boom"}}`, tc.code)
			}))
			defer srv.Close()

			client := newTestClient(srv)

			err := client.Do(t.Context(), http.MethodGet, "/x", nil, nil)

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v (%T), want *APIError", err, err)
			}

			if apiErr.Code != tc.code || apiErr.StatusCode != tc.status {
				t.Errorf("got code=%q status=%d, want code=%q status=%d", apiErr.Code, apiErr.StatusCode, tc.code, tc.status)
			}
		})
	}
}

func TestDoRetriesOnRateLimit(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++

		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"slow down"}}`))

			return
		}

		_, _ = w.Write([]byte(`{"success":true,"data":{"ok":true}}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	var out struct {
		OK bool `json:"ok"`
	}

	if err := client.Do(t.Context(), http.MethodGet, "/x", nil, &out); err != nil {
		t.Fatalf("Do returned error after a retryable 429: %v", err)
	}

	if calls != 2 {
		t.Errorf("calls = %d, want 2 (one 429 then one success)", calls)
	}

	if !out.OK {
		t.Error("response not decoded after retry")
	}
}

func TestDoRateLimitExhausted(t *testing.T) {
	var calls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"nope"}}`))
	}))
	defer srv.Close()

	client := newTestClient(srv)

	err := client.Do(t.Context(), http.MethodGet, "/x", nil, nil)
	if !IsRateLimited(err) {
		t.Fatalf("want RATE_LIMITED after exhausting retries, got %v", err)
	}

	if calls != maxRateLimitRetries+1 {
		t.Errorf("calls = %d, want %d (initial + retries)", calls, maxRateLimitRetries+1)
	}
}

func TestErrorHelpers(t *testing.T) {
	if !IsConflict(&APIError{Code: "CONFLICT"}) {
		t.Error("IsConflict(CONFLICT) = false")
	}

	if IsConflict(&APIError{Code: "NOT_FOUND"}) {
		t.Error("IsConflict(NOT_FOUND) = true")
	}

	if !IsRateLimited(&APIError{Code: "RATE_LIMITED"}) {
		t.Error("IsRateLimited(RATE_LIMITED) = false")
	}

	if IsConflict(nil) || IsRateLimited(nil) || IsNotFound(nil) {
		t.Error("a nil error matched a code helper")
	}
}

func TestRateLimitWait(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		hinted  time.Duration
		want    time.Duration
	}{
		{"server hint honored", 0, 5 * time.Second, 5 * time.Second},
		{"server hint of zero retries instantly", 2, 0, 0},
		{"server hint capped at max", 0, 5 * time.Minute, maxRetryWait},
		{"no hint backs off exponentially (attempt 0)", 0, -1, defaultRetryDelay},
		{"no hint backs off exponentially (attempt 1)", 1, -1, 2 * defaultRetryDelay},
		{"no hint backs off exponentially (attempt 2)", 2, -1, 4 * defaultRetryDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rateLimitWait(tt.attempt, tt.hinted); got != tt.want {
				t.Errorf("rateLimitWait(%d, %v) = %v, want %v", tt.attempt, tt.hinted, got, tt.want)
			}
		})
	}
}
