package provider

import (
	"encoding/json"
	"slices"
	"testing"
)

// TestDomainUnmarshalPointers covers both shapes the API uses for a domain's
// pointers: the spec's array of strings and the live object keyed by pointer
// name. The object form is the regression — it previously failed to decode
// into []string. Object keys come back sorted for deterministic state.
func TestDomainUnmarshalPointers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "array shape (spec)",
			body: `{"domain":"example.com","pointers":["a.com","b.com"]}`,
			want: []string{"a.com", "b.com"},
		},
		{
			name: "object shape (live), keyed by name, sorted",
			body: `{"domain":"example.com","pointers":{"b.com":{"type":"alias"},"a.com":{"type":"redirect"}}}`,
			want: []string{"a.com", "b.com"},
		},
		{
			name: "empty array",
			body: `{"domain":"example.com","pointers":[]}`,
			want: []string{},
		},
		{
			name: "empty object",
			body: `{"domain":"example.com","pointers":{}}`,
			want: []string{},
		},
		{
			name: "null pointers",
			body: `{"domain":"example.com","pointers":null}`,
			want: nil,
		},
		{
			name: "absent pointers",
			body: `{"domain":"example.com"}`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Domain

			if err := json.Unmarshal([]byte(tt.body), &d); err != nil {
				t.Fatalf("Unmarshal returned error: %v", err)
			}

			if d.Domain != "example.com" {
				t.Errorf("Domain = %q, want example.com", d.Domain)
			}

			if !slices.Equal(d.Pointers, tt.want) {
				t.Errorf("Pointers = %#v, want %#v", d.Pointers, tt.want)
			}
		})
	}
}

// TestDomainUnmarshalPointersUnexpectedShape asserts a non-array, non-object
// pointers value is a decode error rather than a silent empty list.
func TestDomainUnmarshalPointersUnexpectedShape(t *testing.T) {
	var d Domain

	err := json.Unmarshal([]byte(`{"domain":"example.com","pointers":"nope"}`), &d)
	if err == nil {
		t.Fatal("expected an error for a string pointers value, got nil")
	}
}
