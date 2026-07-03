package provider

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
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
