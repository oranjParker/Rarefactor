package utils

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		{"invalid-url", ""},
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
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"8.8.8.8", false},
		{"::1", true},
		{"fc00::1", true},
		{"fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", true},
		{"2001:db8::1", false},
	}

	for _, tt := range tests {
		got := IsPrivateIP(net.ParseIP(tt.ip))
		if got != tt.expected {
			t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestBytesCompare(t *testing.T) {
	tests := []struct {
		a, b     []byte
		expected int
	}{
		{[]byte{1, 2}, []byte{1, 2}, 0},
		{[]byte{1, 2}, []byte{1, 3}, -1},
		{[]byte{1, 3}, []byte{1, 2}, 1},
		{[]byte{1, 2}, []byte{1}, 0}, // Different lengths return 0 in current implementation
	}

	for _, tt := range tests {
		got := bytesCompare(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("bytesCompare(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestNewSafeHTTPClient(t *testing.T) {
	cfg := ClientConfig{
		Timeout:       2 * time.Second,
		AllowInternal: false,
	}
	client := NewSafeHTTPClient(cfg)

	if client.Timeout != cfg.Timeout {
		t.Errorf("Expected timeout %v, got %v", cfg.Timeout, client.Timeout)
	}

	// Test blocked connection to private IP
	_, err := client.Get("http://127.0.0.1")
	if err == nil {
		t.Error("Expected error for private IP access, got nil")
	} else if !strings.Contains(err.Error(), "blocked connection to private IP") {
		t.Errorf("Expected SSRF protection error, got: %v", err)
	}

	// Test error on SplitHostPort (invalid URL)
	// NewSafeHTTPClient is called by http.Client.Do, which handles some URL parsing.
	// But our DialContext is called with the address.
	transport := client.Transport.(*http.Transport)
	_, err = transport.DialContext(context.Background(), "tcp", "invalid-addr")
	if err == nil {
		t.Error("Expected error for invalid address, got nil")
	}

	// Test error on LookupIP
	_, err = transport.DialContext(context.Background(), "tcp", "nonexistent.domain.invalid:80")
	if err == nil {
		t.Error("Expected error for nonexistent domain, got nil")
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
