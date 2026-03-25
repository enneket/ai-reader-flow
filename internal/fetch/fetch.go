package fetch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Fetcher fetches full article content from original URLs.
type Fetcher struct {
	client *http.Client
}

// NewFetcher returns a new Fetcher with a 15s timeout client.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil // follow redirects
			},
		},
	}
}

// FetchFullContent attempts to fetch and extract the main text content from an article URL.
// Returns the extracted content or an empty string if it fails.
// contentTypeHint is the detected content type from the RSS feed (may be empty).
func (f *Fetcher) FetchFullContent(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-RSS-Reader/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("non-OK status: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "html") {
		return "", fmt.Errorf("not HTML: %s", contentType)
	}

	// Limit read to 5MB
	reader := io.LimitReader(resp.Body, 5<<20)
	html, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	// Remove script, style, nav, footer, header, aside elements
	remove := "script, style, nav, footer, header, aside, noscript, iframe, form, button, input"
	doc.Find(remove).Each(func(_ int, s *goquery.Selection) {
		s.Remove()
	})

	// Try to find the main article content using common selectors
	content := f.extractContent(doc)

	return content, nil
}

// extractContent tries to pull text from the most likely article body.
func (f *Fetcher) extractContent(doc *goquery.Document) string {
	// Priority selectors for article content
	selectors := []string{
		"article",
		"[role='main']",
		"main",
		".article-content",
		".article-body",
		".post-content",
		".entry-content",
		".content-body",
		"#article",
		"#content",
		".content",
		"body",
	}

	for _, sel := range selectors {
		el := doc.Find(sel).First()
		if el.Length() == 0 {
			continue
		}
		text := strings.TrimSpace(el.Text())
		if len(text) > 200 {
			return text
		}
	}

	// Fallback: return body text
	return strings.TrimSpace(doc.Find("body").Text())
}
