package fetch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchFullContent(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<nav>Navigation</nav>
<article>
<h1>Main Article Title</h1>
<p>This is the main content of the article. It has enough text to be considered valid content for extraction. Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>
</article>
<footer>Footer content</footer>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer server.Close()

	f := NewFetcher()
	content, err := f.FetchFullContent(server.URL)
	if err != nil {
		t.Fatalf("FetchFullContent() error = %v", err)
	}

	if content == "" {
		t.Errorf("FetchFullContent() returned empty content")
	}

	if !strings.Contains(content, "Main Article Title") {
		t.Errorf("FetchFullContent() = %q, expected to contain article title", content)
	}
}

func TestFetchFullContentError(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "non-200 status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    true,
			errContain: "non-OK status: 404",
		},
		{
			name: "non-HTML content",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"error": "not html"}`))
			},
			wantErr:    true,
			errContain: "not HTML",
		},
		{
			name: "connection refused",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Server closes immediately
			},
			wantErr: true,
		},
		{
			name: "empty HTML",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<!DOCTYPE html><html><body></body></html>`))
			},
			wantErr: false, // Should return empty string but no error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "connection refused" {
				// Use a URL that will fail to connect
				f := NewFetcher()
				_, err := f.FetchFullContent("http://localhost:99999")
				if err == nil {
					t.Errorf("FetchFullContent() expected error for connection refused")
				}
				return
			}

			server := httptest.NewServer(tt.handler)
			defer server.Close()

			f := NewFetcher()
			_, err := f.FetchFullContent(server.URL)
			if tt.wantErr {
				if err == nil {
					t.Errorf("FetchFullContent() expected error containing %q", tt.errContain)
				} else if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("FetchFullContent() error = %q, want error containing %q", err.Error(), tt.errContain)
				}
			} else if err != nil {
				t.Errorf("FetchFullContent() unexpected error = %v", err)
			}
		})
	}
}

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantCont string
	}{
		{
			name: "article element",
			html: `<!DOCTYPE html><html><body><article><p>Article content here with enough text to be selected because it exceeds 200 characters threshold. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore.</p></article></body></html>`,
			wantCont: "Article content here",
		},
		{
			name: "main element",
			html: `<!DOCTYPE html><html><body><main><p>Main element content with enough text to be selected. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt.</p></main></body></html>`,
			wantCont: "Main element content",
		},
		{
			name: "role main",
			html: `<!DOCTYPE html><html><body><div role="main"><p>Role main content with enough text here. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod.</p></div></body></html>`,
			wantCont: "Role main content",
		},
		{
			name: "class content",
			html: `<!DOCTYPE html><html><body><div class="content"><p>Class content here with enough text to pass. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod.</p></div></body></html>`,
			wantCont: "Class content here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(tt.html))
			}))
			defer server.Close()

			f := NewFetcher()
			content, err := f.FetchFullContent(server.URL)
			if err != nil {
				t.Fatalf("FetchFullContent() error = %v", err)
			}

			if !strings.Contains(content, tt.wantCont) {
				t.Errorf("FetchFullContent() = %q, want to contain %q", content, tt.wantCont)
			}
		})
	}
}
