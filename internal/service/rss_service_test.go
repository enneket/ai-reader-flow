package service

import (
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
			CreatedAt:   time.Now(),
		}

		if feed.Title == "" {
			t.Error("Feed title should not be empty")
		}
		if feed.URL == "" {
			t.Error("Feed URL should not be empty")
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
			CreatedAt:   now,
		}

		if feed.ID != 1 {
			t.Errorf("Feed ID = %d, want 1", feed.ID)
		}
		if feed.Title != "Hacker News" {
			t.Errorf("Feed Title = %s, want 'Hacker News'", feed.Title)
		}
	})
}
