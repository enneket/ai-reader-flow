package opml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"ai-rss-reader/internal/models"
)

// Export generates an OPML 2.0 XML document from the given feeds.
func Export(feeds []models.Feed) ([]byte, error) {
	type outline struct {
		Text    string `xml:"text,attr"`
		Title   string `xml:"title,attr,omitempty"`
		Type    string `xml:"type,attr,omitempty"`
		XMLURL  string `xml:"xmlUrl,attr,omitempty"`
		HTMLURL string `xml:"htmlUrl,attr,omitempty"`
	}

	type body struct {
		Outline []outline `xml:"outline"`
	}
	type head struct {
		Title string `xml:"title"`
	}
	type opml struct {
		XMLName xml.Name `xml:"opml"`
		Version string   `xml:"version,attr"`
		Head    head     `xml:"head"`
		Body    body     `xml:"body"`
	}

	doc := opml{
		Version: "2.0",
		Head:    head{Title: "AI RSS Reader Subscriptions"},
		Body:    body{Outline: make([]outline, 0, len(feeds))},
	}

	for _, f := range feeds {
		htmlURL := ""
		if f.Description != "" {
			htmlURL = f.Description
		}
		doc.Body.Outline = append(doc.Body.Outline, outline{
			Text:    f.Title,
			Title:   f.Title,
			Type:    "rss",
			XMLURL:  f.URL,
			HTMLURL: htmlURL,
		})
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode opml: %w", err)
	}
	return buf.Bytes(), nil
}

// FeedInfo holds URL and optional title from OPML
type FeedInfo struct {
	URL   string
	Title string
}

// Import parses an OPML document and returns a list of feeds with their titles.
// Only returns feeds that have an xmlUrl attribute set.
func Import(r io.Reader) ([]FeedInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read opml data: %w", err)
	}

	type outline struct {
		XMLURL  string     `xml:"xmlUrl,attr"`
		HTMLURL string     `xml:"htmlUrl,attr,omitempty"`
		Text    string     `xml:"text,attr,omitempty"`
		Title   string     `xml:"title,attr,omitempty"`
		Type    string     `xml:"type,attr,omitempty"`
		Outline []outline  `xml:"outline"`
	}
	type body struct {
		Outline []outline `xml:"outline"`
	}
	type head struct {
		Title string `xml:"title"`
	}
	type opml struct {
		Version string `xml:"version,attr"`
		Head    head   `xml:"head"`
		Body    body   `xml:"body"`
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false

	var doc opml
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode opml: %w", err)
	}

	var feeds []FeedInfo
	var collect func([]outline)
	collect = func(outlines []outline) {
		for _, o := range outlines {
			if o.XMLURL != "" {
				title := o.Text
				if title == "" {
					title = o.Title
				}
				feeds = append(feeds, FeedInfo{
					URL:   strings.TrimSpace(o.XMLURL),
					Title: title,
				})
			}
			if len(o.Outline) > 0 {
				collect(o.Outline)
			}
		}
	}
	collect(doc.Body.Outline)

	return feeds, nil
}
