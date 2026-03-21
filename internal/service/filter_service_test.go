package service

import (
	"ai-rss-reader/internal/models"
	"testing"
)

func TestMatchKeyword(t *testing.T) {
	s := &FilterService{}

	tests := []struct {
		name     string
		article  *models.Article
		keyword  string
		expected bool
	}{
		{
			name: "keyword match in title",
			article: &models.Article{
				Title:   "Go Programming Tutorial",
				Content: "Learn Go language",
			},
			keyword:  "Go",
			expected: true,
		},
		{
			name: "keyword match in content",
			article: &models.Article{
				Title:   "Programming Tutorial",
				Content: "Learn the Go language",
			},
			keyword:  "Go",
			expected: true,
		},
		{
			name: "keyword case insensitive",
			article: &models.Article{
				Title:   "Python Tutorial",
				Content: "Learn Python programming",
			},
			keyword:  "python",
			expected: true,
		},
		{
			name: "keyword not found",
			article: &models.Article{
				Title:   "Rust Tutorial",
				Content: "Learn Rust programming",
			},
			keyword:  "Go",
			expected: false,
		},
		{
			name: "empty keyword matches all (strings.Contains behavior)",
			article: &models.Article{
				Title:   "Any Title",
				Content: "Any content",
			},
			keyword:  "",
			expected: true, // strings.Contains returns true for empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.matchKeyword(tt.article, tt.keyword)
			if result != tt.expected {
				t.Errorf("matchKeyword() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMatchSource(t *testing.T) {
	s := &FilterService{}

	tests := []struct {
		name     string
		article  *models.Article
		source   string
		expected bool
	}{
		{
			name: "source match",
			article: &models.Article{
				Author: "John Doe",
			},
			source:   "John",
			expected: true,
		},
		{
			name: "source case insensitive",
			article: &models.Article{
				Author: "JOHN DOE",
			},
			source:   "john",
			expected: true,
		},
		{
			name: "source not found",
			article: &models.Article{
				Author: "Jane Doe",
			},
			source:   "John",
			expected: false,
		},
		{
			name: "empty author",
			article: &models.Article{
				Author: "",
			},
			source:   "John",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.matchSource(tt.article, tt.source)
			if result != tt.expected {
				t.Errorf("matchSource() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluateRule(t *testing.T) {
	s := &FilterService{}

	tests := []struct {
		name     string
		article  *models.Article
		rule     *models.FilterRule
		expected bool
	}{
		{
			name: "keyword rule matches",
			article: &models.Article{
				Title:   "Go Tutorial",
				Content: "Learn Go",
			},
			rule: &models.FilterRule{
				Type:  "keyword",
				Value: "Go",
			},
			expected: true,
		},
		{
			name: "keyword rule no match",
			article: &models.Article{
				Title:   "Rust Tutorial",
				Content: "Learn Rust",
			},
			rule: &models.FilterRule{
				Type:  "keyword",
				Value: "Go",
			},
			expected: false,
		},
		{
			name: "source rule matches",
			article: &models.Article{
				Author: "John Doe",
			},
			rule: &models.FilterRule{
				Type:  "source",
				Value: "John",
			},
			expected: true,
		},
		{
			name: "unknown rule type defaults to true",
			article: &models.Article{
				Title: "Any Title",
			},
			rule: &models.FilterRule{
				Type:  "unknown",
				Value: "anything",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.evaluateRule(tt.article, tt.rule)
			if err != nil {
				t.Errorf("evaluateRule() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("evaluateRule() = %v, want %v", result, tt.expected)
			}
		})
	}
}
