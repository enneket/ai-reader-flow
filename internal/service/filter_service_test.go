package service

import (
	"ai-rss-reader/internal/models"
	"math"
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

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float64
		approx   bool // if true, use approximate comparison
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
			approx:   false,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{0, 1, 0},
			expected: 0.0,
			approx:   false,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{-1, 0, 0},
			expected: -1.0,
			approx:   false,
		},
		{
			name:     "nil a",
			a:        nil,
			b:        []float32{1, 0, 0},
			expected: 0.0,
			approx:   false,
		},
		{
			name:     "nil b",
			a:        []float32{1, 0, 0},
			b:        nil,
			expected: 0.0,
			approx:   false,
		},
		{
			name:     "empty a",
			a:        []float32{},
			b:        []float32{1, 0, 0},
			expected: 0.0,
			approx:   false,
		},
		{
			name:     "mismatched length",
			a:        []float32{1, 0},
			b:        []float32{1, 0, 0},
			expected: 0.0,
			approx:   false,
		},
		{
			name:     "3d vectors similar",
			a:        []float32{0.8, 0.2, 0.1},
			b:        []float32{0.9, 0.1, 0.2},
			expected: 0.986,
			approx:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if tt.approx {
				if math.Abs(result-tt.expected) > 0.01 {
					t.Errorf("CosineSimilarity() = %v, want ~%v", result, tt.expected)
				}
			} else {
				if result != tt.expected {
					t.Errorf("CosineSimilarity() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestScoreTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected int
	}{
		{"empty", "", 0},
		{"too short 1 char", "A", 0},
		{"too short 3 chars", "Hi!", 0},
		{"short 5 chars", "Hello", 5},
		{"medium 50 chars no bonus", "this is a medium length title without numbers or caps", 20},
		{"medium with number", "Go 1.22 Released with Major New Features", 25},
		{"medium with uppercase", "RUST Programming Language Update", 25},
		{"long 120 chars", string(make([]byte, 120)), 25},
		{"too long 160 chars", string(make([]byte, 160)), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoreTitle(tt.title)
			if result != tt.expected {
				t.Errorf("scoreTitle(%q) = %v, want %v", tt.title, result, tt.expected)
			}
		})
	}
}

func TestScoreLength(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		expected int
	}{
		{"empty", 0, 0},
		{"very short 100", 100, 0},
		{"short 500", 500, 5},
		{"medium 999", 999, 5},
		{"medium 1000", 1000, 10},
		{"medium 1999", 1999, 10},
		{"long 2000", 2000, 15},
		{"long 5000", 5000, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := string(make([]byte, tt.length))
			result := scoreLength(content)
			if result != tt.expected {
				t.Errorf("scoreLength(len=%d) = %v, want %v", tt.length, result, tt.expected)
			}
		})
	}
}

func TestQualityScore(t *testing.T) {
	s := &FilterService{}

	tests := []struct {
		name          string
		title         string
		contentLength int
		minExpected   int
		maxExpected   int
	}{
		{"short title + short content", "Hi", 100, 0, 5},
		{"medium title + medium content", "Understanding Rust Ownership and Borrowing", 1500, 20, 40},
		{"long title + long content", "Go 1.22 Released: Major Performance Improvements and New Features in the Latest Update", 3000, 30, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := &models.Article{
				Title:   tt.title,
				Content: string(make([]byte, tt.contentLength)),
			}
			result := s.QualityScore(article)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("QualityScore() = %v, want between %v and %v", result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestNoMoreEmbeddingTests(t *testing.T) {
	// All embedding-related tests removed
}
