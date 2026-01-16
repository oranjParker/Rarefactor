package crawler

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oranjParker/Rarefactor/internal/utils"
)

func TestCrawler_Fetch(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><a href="/link1">Test</a></body></html>`))
	}))
	defer mockServer.Close()

	cfg := utils.ClientConfig{
		Timeout:       5 * time.Second,
		AllowInternal: true,
	}
	client := utils.NewSafeHTTPClient(cfg)

	resp, err := client.Get(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestCrawler_SSRFProtection(t *testing.T) {
	cfg := utils.ClientConfig{
		Timeout:       2 * time.Second,
		AllowInternal: false,
	}
	client := utils.NewSafeHTTPClient(cfg)

	tests := []struct {
		name string
		url  string
	}{
		{"IPv4 Loopback", "http://127.0.0.1:8080"},
		{"IPv6 Loopback", "http://[::1]:8080"},
		{"Private Class A", "http://10.0.0.1"},
		{"Private Class B", "http://172.16.0.1"},
		{"Private Class C", "http://192.168.1.1"},
		{"AWS Metadata", "http://169.254.169.254"},
		{"IPv6 ULA", "http://[fc00::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Get(tt.url)
			if err == nil {
				t.Errorf("%s: Expected error when accessing private IP, but got none", tt.name)
			} else {
				t.Logf("%s: Successfully blocked: %v", tt.name, err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip        string
		isPrivate bool
	}{
		{"8.8.8.8", false},
		{"127.0.0.1", true},
		{"10.0.0.5", true},
		{"192.168.1.100", true},
		{"172.20.0.1", true},
		{"1.1.1.1", false},
		{"[::1]", true},
		{"[fc00::1]", true},
		{"[2001:4860:4860::8888]", false},
		{"169.254.169.254", true},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			parsed := net.ParseIP(tt.ip)
			if parsed == nil {
				host, _, _ := net.SplitHostPort(fmt.Sprintf("%s:80", tt.ip))
				parsed = net.ParseIP(host)
			}

			if res := utils.IsPrivateIP(parsed); res != tt.isPrivate {
				t.Errorf("IsPrivateIP(%s) = %v; want %v", tt.ip, res, tt.isPrivate)
			}
		})
	}
}
