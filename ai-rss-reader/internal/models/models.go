package models

import "time"

type Feed struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	IconURL     string    `json:"icon_url"`
	LastFetched time.Time `json:"last_fetched"`
	CreatedAt   time.Time `json:"created_at"`
}

type Article struct {
	ID         int64     `json:"id"`
	FeedID     int64     `json:"feed_id"`
	Title      string    `json:"title"`
	Link       string    `json:"link"`
	Content    string    `json:"content"`
	Summary    string    `json:"summary"`
	Author     string    `json:"author"`
	Published  time.Time `json:"published"`
	IsFiltered bool      `json:"is_filtered"`
	IsSaved    bool      `json:"is_saved"`
	CreatedAt  time.Time `json:"created_at"`
}

type FilterRule struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // keyword, source, ai_score
	Value     string `json:"value"`
	Action    string `json:"action"` // include, exclude
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
}

type Note struct {
	ID         int64  `json:"id"`
	ArticleID  int64  `json:"article_id"`
	FilePath   string `json:"file_path"`
	Title      string `json:"title"`
	CreatedAt  string `json:"created_at"`
}

type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AIProviderConfig holds AI provider configuration
type AIProviderConfig struct {
	Provider   string `json:"provider"`   // openai, claude, ollama
	APIKey     string `json:"api_key"`
	BaseURL    string `json:"base_url"`
	Model      string `json:"model"`
	MaxTokens  int    `json:"max_tokens"`
}

// AppState holds the application state for frontend
type AppState struct {
	Feeds         []Feed         `json:"feeds"`
	Articles      []Article      `json:"articles"`
	FilterRules   []FilterRule   `json:"filter_rules"`
	Notes         []Note         `json:"notes"`
	AIConfig      AIProviderConfig `json:"ai_config"`
	FilterMode    string         `json:"filter_mode"` // all, filtered, saved
}
