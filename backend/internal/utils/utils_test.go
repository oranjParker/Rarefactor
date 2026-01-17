package utils

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSanitizeUTF8(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"Hello \xff World", "Hello  World"},
	}

	for _, tt := range tests {
		got := SanitizeUTF8(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeUTF8(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetBaseDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://google.com", "google.com"},
		{"https://blog.google.co.uk/path", "google.co.uk"},
		{"http://localhost:8080", "localhost"},
	}

	for _, tt := range tests {
		got, _ := GetBaseDomain(tt.url)
		if got != tt.expected {
			t.Errorf("GetBaseDomain(%q) = %q, want %q", tt.url, got, tt.expected)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"8.8.8.8", false},
		{"::1", true},
	}

	for _, tt := range tests {
		got := IsPrivateIP(net.ParseIP(tt.ip))
		if got != tt.expected {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestAllowCORS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	corsHandler := AllowCORS(handler)

	req := httptest.NewRequest("OPTIONS", "http://example.com", nil)
	w := httptest.NewRecorder()
	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set")
	}
}
