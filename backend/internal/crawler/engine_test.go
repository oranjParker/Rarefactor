package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/oranjParker/Rarefactor/internal/utils"
	"github.com/redis/go-redis/v9"
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

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base     string
		rel      string
		expected string
	}{
		{"http://example.com", "/a", "http://example.com/a"},
		{"http://example.com/b", "c", "http://example.com/c"},
		{"http://example.com", "http://other.com", "http://other.com"},
		{"invalid-base", "/a", ""},
		{"http://example.com", " :invalid-rel", ""},
		{"http://example.com", strings.Repeat("a", 3000), ""},
	}

	for _, tt := range tests {
		got := resolveURL(tt.base, tt.rel)
		if got != tt.expected {
			t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.rel, got, tt.expected)
		}
	}
}

func TestIsLikelyHTML(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://example.com/index.html", true},
		{"http://example.com/image.jpg", false},
		{"http://example.com/doc.pdf", false},
		{"http://example.com/path", true},
		{"ftp://example.com", false},
	}

	for _, tt := range tests {
		got := isLikelyHTML(tt.url)
		if got != tt.expected {
			t.Errorf("isLikelyHTML(%q) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

func TestEngine_Run_Targeted(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="http://other.com">Other</a><a href="/page1">Same</a></body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body>Page 1</body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	e := NewEngine(nil, 1, 0, nil, nil, nil)
	e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})
	mock := &mockStorage{}
	e.storage = mock

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	e.Run(ctx, server.URL, "test", 2, "targeted")

	for _, doc := range mock.persistedDocs {
		if strings.Contains(doc.url, "other.com") {
			t.Errorf("Targeted crawl should not have persisted other.com, got %s", doc.url)
		}
	}
}

func TestEngine_IsAllowed_More(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	e := NewEngine(nil, 1, 0, nil, nil, nil)
	e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})

	// 404 Case
	allowed, _ := e.isAllowed(context.Background(), server.URL+"/any")
	if !allowed {
		t.Error("Expected allowed=true for 404 robots.txt")
	}

	// Invalid URL
	allowed, _ = e.isAllowed(context.Background(), "::invalid")
	if allowed {
		t.Error("Expected allowed=false for invalid URL (fail-closed)")
	}
}

func TestEngine_IsAllowed_Redis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e := NewEngine(nil, 1, 0, rdb, nil, nil)
	e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})

	ctx := context.Background()
	targetHost := "example.com"
	targetURL := "http://" + targetHost + "/allowed"

	// 1. Initially not in redis, should fetch
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "User-agent: *\nDisallow: /blocked")
	}))
	defer server.Close()

	// Adjust targetURL to use test server
	u, _ := url.Parse(server.URL)
	targetURL = server.URL + "/allowed"

	allowed, err := e.isAllowed(ctx, targetURL)
	if err != nil || !allowed {
		t.Errorf("Expected allowed, got %v, err: %v", allowed, err)
	}

	// 2. Should be in redis now
	cacheKey := fmt.Sprintf("robots:%s", u.Host)
	val, err := rdb.Get(ctx, cacheKey).Result()
	if err != nil || !strings.Contains(val, "Disallow: /blocked") {
		t.Errorf("Robots data not cached in redis correctly: %v", val)
	}

	// 3. Test disallowed path from cache
	allowed, _ = e.isAllowed(ctx, server.URL+"/blocked")
	if allowed {
		t.Error("Expected disallowed for /blocked")
	}

	// 4. Test 404 caching
	mr.FlushAll()
	server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server404.Close()

	u404, _ := url.Parse(server404.URL)
	allowed, _ = e.isAllowed(ctx, server404.URL+"/any")
	if !allowed {
		t.Error("Expected allowed for 404 robots.txt")
	}

	cacheKey404 := fmt.Sprintf("robots:%s", u404.Host)
	val404, _ := rdb.Get(ctx, cacheKey404).Result()
	if val404 != "" {
		t.Errorf("Expected empty string in cache for 404, got %q", val404)
	}
}

func TestEngine_PersistDocument_NoDB(t *testing.T) {
	e := NewEngine(nil, 1, 0, nil, nil, nil)
	err := e.PersistDocument(context.Background(), "http://example.com", "Title", "Content", "ns")
	if err != nil {
		t.Errorf("Expected nil error when dbPool is nil, got %v", err)
	}
}

func TestEngine_Run_ModeBroad(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><a href="/page1">Link</a></body></html>`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	e := NewEngine(nil, 1, 0, nil, nil, nil)
	e.httpClient = utils.NewSafeHTTPClient(utils.ClientConfig{AllowInternal: true})
	mock := &mockStorage{}
	e.storage = mock

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	e.Run(ctx, server.URL, "test", 10, "broad")
	// broad mode should force maxDepth to 2
	for _, doc := range mock.persistedDocs {
		// Just verify it ran
		if doc.url == server.URL {
			return
		}
	}
	t.Error("Engine did not run in broad mode")
}

func TestEngine_Run_InvalidSeed(t *testing.T) {
	e := NewEngine(nil, 1, 0, nil, nil, nil)
	// Targeted mode with invalid seed URL
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	e.Run(ctx, ":invalid", "test", 10, "targeted")
	// Should return early
}
