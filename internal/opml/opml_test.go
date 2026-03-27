package opml

import (
	"strings"
	"testing"

	"ai-rss-reader/internal/models"
)

func TestExport(t *testing.T) {
	tests := []struct {
		name   string
		feeds  []models.Feed
		checks func(t *testing.T, data []byte)
	}{
		{
			name:   "empty feeds",
			feeds:  []models.Feed{},
			checks: func(t *testing.T, data []byte) {
				if !strings.Contains(string(data), "<?xml") {
					t.Errorf("expected xml declaration")
				}
				if !strings.Contains(string(data), "<opml") {
					t.Errorf("expected <opml> element")
				}
				if !strings.Contains(string(data), "version=\"2.0\"") {
					t.Errorf("expected version=\"2.0\"")
				}
				// Empty feeds still produce an OPML with empty body outline
				if !strings.Contains(string(data), "<body>") {
					t.Errorf("expected <body> element")
				}
			},
		},
		{
			name: "single feed",
			feeds: []models.Feed{
				{Title: "Test Feed", URL: "https://example.com/feed.xml", Description: "https://example.com"},
			},
			checks: func(t *testing.T, data []byte) {
				s := string(data)
				if !strings.Contains(s, "Test Feed") {
					t.Errorf("expected feed title in output")
				}
				if !strings.Contains(s, "https://example.com/feed.xml") {
					t.Errorf("expected feed URL in output")
				}
				if !strings.Contains(s, "type=\"rss\"") {
					t.Errorf("expected type=\"rss\" attribute")
				}
			},
		},
		{
			name: "multiple feeds",
			feeds: []models.Feed{
				{Title: "Feed 1", URL: "https://example.com/1.xml"},
				{Title: "Feed 2", URL: "https://example.com/2.xml"},
				{Title: "Feed 3", URL: "https://example.com/3.xml"},
			},
			checks: func(t *testing.T, data []byte) {
				s := string(data)
				if !strings.Contains(s, "Feed 1") {
					t.Errorf("expected Feed 1 in output")
				}
				if !strings.Contains(s, "Feed 2") {
					t.Errorf("expected Feed 2 in output")
				}
				if !strings.Contains(s, "Feed 3") {
					t.Errorf("expected Feed 3 in output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Export(tt.feeds)
			if err != nil {
				t.Fatalf("Export() error = %v", err)
			}
			tt.checks(t, data)
		})
	}
}

func TestImport(t *testing.T) {
	tests := []struct {
		name      string
		opml      string
		wantURLs  []string
		wantError bool
	}{
		{
			name: "valid OPML with single feed",
			opml: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" text="Example" xmlUrl="https://example.com/feed.xml"/>
  </body>
</opml>`,
			wantURLs: []string{"https://example.com/feed.xml"},
		},
		{
			name: "valid OPML with multiple feeds",
			opml: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" text="Feed 1" xmlUrl="https://example.com/1.xml"/>
    <outline type="rss" text="Feed 2" xmlUrl="https://example.com/2.xml"/>
  </body>
</opml>`,
			wantURLs: []string{"https://example.com/1.xml", "https://example.com/2.xml"},
		},
		{
			name: "nested outlines",
			opml: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline text="Group">
      <outline type="rss" text="Nested Feed" xmlUrl="https://example.com/nested.xml"/>
    </outline>
    <outline type="rss" text="Top Level" xmlUrl="https://example.com/top.xml"/>
  </body>
</opml>`,
			wantURLs: []string{"https://example.com/nested.xml", "https://example.com/top.xml"},
		},
		{
			name: "outline with no xmlUrl",
			opml: `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head><title>Test</title></head>
  <body>
    <outline type="rss" text="No URL" xmlUrl=""/>
    <outline type="rss" text="Valid" xmlUrl="https://example.com/valid.xml"/>
  </body>
</opml>`,
			wantURLs: []string{"https://example.com/valid.xml"},
		},
		{
			name:      "empty OPML",
			opml:      `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><head><title>Test</title></head><body></body></opml>`,
			wantURLs:  []string{},
		},
		{
			name:      "invalid XML",
			opml:      `<not valid xml`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls, err := Import(strings.NewReader(tt.opml))
			if tt.wantError {
				if err == nil {
					t.Errorf("Import() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Import() error = %v", err)
			}
			if len(urls) != len(tt.wantURLs) {
				t.Errorf("Import() got %d URLs, want %d", len(urls), len(tt.wantURLs))
				return
			}
			for i, want := range tt.wantURLs {
				if urls[i] != want {
					t.Errorf("Import()[%d] = %q, want %q", i, urls[i], want)
				}
			}
		})
	}
}
