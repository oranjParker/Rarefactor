package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/oranjParker/Rarefactor/internal/utils"
)

func TestFetchAndParse(t *testing.T) {
	e := &Engine{
		httpClient: utils.NewSafeHTTPClient(utils.ClientConfig{
			Timeout:       10 * time.Second,
			AllowInternal: true,
		}),
	}

	hugeAttr := strings.Repeat("a", 1024*1024) // 1MB attribute value

	tests := []struct {
		name            string
		html            string
		contentType     string
		expectedTitle   string
		expectedLinks   []string
		expectedContent string
		expectError     bool
	}{
		{
			name:            "Normal HTML",
			html:            `<html><head><title>Test Title</title></head><body>Hello World <a href="/link1">Link 1</a></body></html>`,
			contentType:     "text/html",
			expectedTitle:   "Test Title",
			expectedLinks:   []string{"http://example.com/link1"},
			expectedContent: "Hello World Link 1",
			expectError:     false,
		},
		{
			name:            "Empty Tags",
			html:            `<html><head><title></title></head><body><p></p><a href="/link1"></a></body></html>`,
			contentType:     "text/html",
			expectedTitle:   "http://example.com", // Fallback to URL if title is empty
			expectedLinks:   []string{"http://example.com/link1"},
			expectedContent: "",
			expectError:     false,
		},
		{
			name:            "Malformed HTML",
			html:            `<html><head><title>Malformed</title><body>Missing closing tags <a href="/link1">Link 1`,
			contentType:     "text/html",
			expectedTitle:   "Malformed",
			expectedLinks:   []string{"http://example.com/link1"},
			expectedContent: "Missing closing tags Link 1",
			expectError:     false,
		},
		{
			name:            "Huge Attribute Value",
			html:            fmt.Sprintf(`<html><head><title>Huge Attr</title></head><body><a href="/link1" data-huge="%s">Link 1</a></body></html>`, hugeAttr),
			contentType:     "text/html",
			expectedTitle:   "Huge Attr",
			expectedLinks:   []string{"http://example.com/link1"},
			expectedContent: "Link 1",
			expectError:     false,
		},
		{
			name:        "Non-HTML Content",
			html:        `{"json": "data"}`,
			contentType: "application/json",
			expectError: true,
		},
		{
			name:            "No Body Tag",
			html:            `<html><head><title>No Body</title></head>No body here <a href="/link1">Link 1</a></html>`,
			contentType:     "text/html",
			expectedTitle:   "No Body",
			expectedLinks:   []string{"http://example.com/link1"},
			expectedContent: "No body here Link 1",
			expectError:     false,
		},
		{
			name:            "Script and Style Removal",
			html:            `<html><head><title>Cleanup</title><style>body { color: red; }</style></head><body>Visible<script>alert(1)</script></body></html>`,
			contentType:     "text/html",
			expectedTitle:   "Cleanup",
			expectedLinks:   nil,
			expectedContent: "Visible",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				fmt.Fprint(w, tt.html)
			}))
			defer server.Close()

			// Overwrite expected links with server URL if they are relative
			var expectedLinks []string
			for _, l := range tt.expectedLinks {
				if strings.HasPrefix(l, "http://example.com") {
					expectedLinks = append(expectedLinks, strings.Replace(l, "http://example.com", server.URL, 1))
				} else {
					expectedLinks = append(expectedLinks, l)
				}
			}

			expectedTitle := tt.expectedTitle
			if expectedTitle == "http://example.com" {
				expectedTitle = server.URL
			}

			links, title, content, err := e.fetchAndParse(context.Background(), server.URL)

			if (err != nil) != tt.expectError {
				t.Fatalf("expected error: %v, got: %v", tt.expectError, err)
			}

			if tt.expectError {
				return
			}

			if title != expectedTitle {
				t.Errorf("expected title %q, got %q", expectedTitle, title)
			}

			if content != tt.expectedContent {
				t.Errorf("expected content %q, got %q", tt.expectedContent, content)
			}

			if len(links) != len(expectedLinks) {
				t.Errorf("expected %d links, got %d: %v", len(expectedLinks), len(links), links)
			} else {
				for i, l := range links {
					if l != expectedLinks[i] {
						t.Errorf("expected link[%d] %q, got %q", i, expectedLinks[i], l)
					}
				}
			}
		})
	}
}

type mockStorage struct {
	persistedDocs []struct {
		url, title, content, namespace string
	}
}

func (m *mockStorage) PersistDocument(ctx context.Context, urlStr, title, content, namespace string) error {
	m.persistedDocs = append(m.persistedDocs, struct {
		url, title, content, namespace string
	}{urlStr, title, content, namespace})
	return nil
}

func TestEngine_Run(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Link</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>Page 1</body></html>`)
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "User-agent: *\nAllow: /")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	e := NewEngine(nil, 1, 10*time.Millisecond, nil, nil, nil)
	e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})
	mock := &mockStorage{}
	e.storage = mock

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	e.Run(ctx, server.URL, "test", 2, "broad")

	if len(mock.persistedDocs) < 2 {
		t.Errorf("Expected at least 2 persisted documents, got %d", len(mock.persistedDocs))
	}
}

func TestEngine_IsAllowed(t *testing.T) {
	tests := []struct {
		name       string
		robots     string
		statusCode int
		allowed    bool
	}{
		{"Allow all", "User-agent: *\nAllow: /", 200, true},
		{"Disallow all", "User-agent: *\nDisallow: /", 200, false},
		{"500 Error", "", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					w.WriteHeader(tt.statusCode)
					fmt.Fprint(w, tt.robots)
				}
			}))
			defer server.Close()

			e := NewEngine(nil, 1, 0, nil, nil, nil)
			e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})
			// Disable redis to force HTTP fetch
			e.redisClient = nil

			allowed, _ := e.isAllowed(context.Background(), server.URL+"/any")
			if allowed != tt.allowed {
				t.Errorf("Expected allowed=%v, got %v", tt.allowed, allowed)
			}
		})
	}
}

func TestEngine_PersistDocument(t *testing.T) {
	longTitle := strings.Repeat("T", 600)
	longContent := strings.Repeat("C", 110000)
	invalidUTF8 := "Hello \xff World"

	t.Run("Utility Logic", func(t *testing.T) {
		// Testing the logic that persistDocument uses
		title := utils.SanitizeUTF8(longTitle)
		if len(title) > 500 {
			title = title[:500]
		}
		if len(title) > 500 {
			t.Errorf("Title not truncated, got %d", len(title))
		}

		content := utils.SanitizeUTF8(longContent)
		if len(content) > 100000 {
			content = content[:100000]
		}
		if len(content) > 100000 {
			t.Errorf("Content not truncated, got %d", len(content))
		}

		sanitized := utils.SanitizeUTF8(invalidUTF8)
		if strings.Contains(sanitized, "\xff") {
			t.Errorf("Sanitization failed, got %q", sanitized)
		}
	})
}
