package service

import (
	"fmt"
	"testing"
	"time"

	"ai-rss-reader/internal/models"
)

func TestRSSServiceHelpers(t *testing.T) {
	// Test the feed model creation without network/database
	t.Run("feed model creation", func(t *testing.T) {
		feed := &models.Feed{
			Title:       "Test Feed",
			URL:         "https://example.com/feed",
			Description: "A test feed",
			IconURL:     "https://example.com/icon.png",
			LastFetched: time.Now(),
			IsDead:      false,
			CreatedAt:   time.Now(),
		}

		if feed.Title == "" {
			t.Error("Feed title should not be empty")
		}
		if feed.URL == "" {
			t.Error("Feed URL should not be empty")
		}
		if feed.IsDead {
			t.Error("New feed should not be dead")
		}
	})

	t.Run("article model creation", func(t *testing.T) {
		article := &models.Article{
			FeedID:     1,
			Title:      "Test Article",
			Link:       "https://example.com/article",
			Content:    "Article content here",
			Summary:    "Article summary",
			Author:     "Test Author",
			Published:  time.Now(),
			IsFiltered: false,
			IsSaved:    false,
			Status:     "unread",
			CreatedAt:  time.Now(),
		}

		if article.Title == "" {
			t.Error("Article title should not be empty")
		}
		if article.Link == "" {
			t.Error("Article link should not be empty")
		}
		if article.FeedID == 0 {
			t.Error("Article FeedID should not be zero")
		}
		if article.Status != "unread" {
			t.Errorf("Article Status = %s, want 'unread'", article.Status)
		}
	})
}

func TestArticleModel(t *testing.T) {
	t.Run("default filter state is false", func(t *testing.T) {
		article := &models.Article{}
		if article.IsFiltered {
			t.Error("Default IsFiltered should be false")
		}
	})

	t.Run("default saved state is false", func(t *testing.T) {
		article := &models.Article{}
		if article.IsSaved {
			t.Error("Default IsSaved should be false")
		}
	})

	t.Run("default status is unread", func(t *testing.T) {
		article := &models.Article{}
		if article.Status != "" && article.Status != "unread" {
			t.Errorf("Default Status = %s, want 'unread' or empty", article.Status)
		}
	})
}

func TestFeedModel(t *testing.T) {
	t.Run("feed with all fields", func(t *testing.T) {
		now := time.Now()
		feed := &models.Feed{
			ID:          1,
			Title:       "Hacker News",
			URL:         "https://news.ycombinator.com/rss",
			Description: "Hacker News RSS Feed",
			IconURL:     "",
			LastFetched: now,
			IsDead:      false,
			CreatedAt:   now,
		}

		if feed.ID != 1 {
			t.Errorf("Feed ID = %d, want 1", feed.ID)
		}
		if feed.Title != "Hacker News" {
			t.Errorf("Feed Title = %s, want 'Hacker News'", feed.Title)
		}
		if feed.IsDead {
			t.Error("Feed should not be dead by default")
		}
	})
}

func TestIsHTTPNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "404 in error string",
			err:      fmt.Errorf("GET https://example.com: 404 Not Found"),
			expected: true,
		},
		{
			name:     "410 Gone in error string",
			err:      fmt.Errorf("GET https://example.com: 410 Gone"),
			expected: true,
		},
		{
			name:     "StatusCode 404",
			err:      fmt.Errorf("StatusCode: 404"),
			expected: true,
		},
		{
			name:     "StatusCode 410",
			err:      fmt.Errorf("StatusCode: 410"),
			expected: true,
		},
		{
			name:     "not found string",
			err:      fmt.Errorf("feed not found"),
			expected: true,
		},
		{
			name:     "Gone string",
			err:      fmt.Errorf("resource Gone"),
			expected: true,
		},
		{
			name:     "random error returns false",
			err:      fmt.Errorf("connection timeout"),
			expected: false,
		},
		{
			name:     "server error returns false",
			err:      fmt.Errorf("500 Internal Server Error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTTPNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("isHTTPNotFound(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestArticleStatusValues(t *testing.T) {
	validStatuses := []string{"unread", "accepted", "rejected", "snoozed"}

	for _, status := range validStatuses {
		article := &models.Article{Status: status}
		if article.Status != status {
			t.Errorf("Article Status = %s, want %s", article.Status, status)
		}
	}
}
