package models

import (
	"testing"
	"time"
)

func TestFeedModel(t *testing.T) {
	t.Run("create feed", func(t *testing.T) {
		feed := &Feed{
			ID:          1,
			Title:       "Test Feed",
			URL:         "https://test.com/feed",
			Description: "A test feed",
			IconURL:     "https://test.com/icon.png",
			LastFetched: time.Now(),
			CreatedAt:   time.Now(),
		}

		if feed.ID != 1 {
			t.Errorf("ID = %d, want 1", feed.ID)
		}
		if feed.Title == "" {
			t.Error("Title should not be empty")
		}
	})

	t.Run("feed URL validation", func(t *testing.T) {
		feed := &Feed{
			URL: "not-a-url",
		}
		// URL is not validated in model, just check it's stored
		if feed.URL != "not-a-url" {
			t.Error("URL should be stored as is")
		}
	})
}

func TestArticleModel(t *testing.T) {
	t.Run("create article", func(t *testing.T) {
		now := time.Now()
		article := &Article{
			ID:         1,
			FeedID:     1,
			Title:      "Test Article",
			Link:       "https://test.com/article",
			Content:    "Article content",
			Summary:    "Summary",
			Author:     "Author Name",
			Published:  now,
			IsFiltered: false,
			IsSaved:    false,
			CreatedAt:  now,
		}

		if article.ID != 1 {
			t.Errorf("ID = %d, want 1", article.ID)
		}
		if article.Title == "" {
			t.Error("Title should not be empty")
		}
		if article.IsFiltered {
			t.Error("IsFiltered should be false by default")
		}
	})

	t.Run("article is not filtered by default", func(t *testing.T) {
		article := &Article{}
		if article.IsFiltered {
			t.Error("IsFiltered should default to false")
		}
	})

	t.Run("article is not saved by default", func(t *testing.T) {
		article := &Article{}
		if article.IsSaved {
			t.Error("IsSaved should default to false")
		}
	})
}

func TestNoteModel(t *testing.T) {
	t.Run("create note", func(t *testing.T) {
		note := &Note{
			ID:        1,
			ArticleID: 10,
			FilePath:  "/path/to/note.md",
			Title:     "Test Note",
		}

		if note.ID != 1 {
			t.Errorf("ID = %d, want 1", note.ID)
		}
		if note.ArticleID != 10 {
			t.Errorf("ArticleID = %d, want 10", note.ArticleID)
		}
	})
}

func TestAIProviderConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := &AIProviderConfig{
			Provider:  "openai",
			APIKey:    "test-key",
			BaseURL:   "https://api.openai.com/v1",
			Model:     "gpt-3.5-turbo",
			MaxTokens: 500,
		}

		if config.Provider != "openai" {
			t.Errorf("Provider = %s, want 'openai'", config.Provider)
		}
		if config.MaxTokens != 500 {
			t.Errorf("MaxTokens = %d, want 500", config.MaxTokens)
		}
	})
}

func TestAppState(t *testing.T) {
	t.Run("empty app state", func(t *testing.T) {
		state := &AppState{}

		if state.Feeds != nil {
			t.Error("Feeds should be nil by default")
		}
		if state.Articles != nil {
			t.Error("Articles should be nil by default")
		}
	})

	t.Run("app state with data", func(t *testing.T) {
		state := &AppState{
			Feeds:       []Feed{{ID: 1, Title: "Feed 1"}},
			Articles:    []Article{{ID: 1, Title: "Article 1"}},
			FilterMode: "all",
		}

		if len(state.Feeds) != 1 {
			t.Errorf("len(Feeds) = %d, want 1", len(state.Feeds))
		}
		if state.FilterMode != "all" {
			t.Errorf("FilterMode = %s, want 'all'", state.FilterMode)
		}
	})
}
