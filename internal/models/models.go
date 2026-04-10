package models

import "time"

type Feed struct {
	ID                int64     `json:"id"`
	Title             string    `json:"title"`
	URL               string    `json:"url"`
	Description       string    `json:"description"`
	IconURL           string    `json:"icon_url"`
	LastFetched       time.Time `json:"last_fetched"`
	IsDead            bool      `json:"is_dead"` // true if feed returned 404/410
	CreatedAt         time.Time `json:"created_at"`
	Group             string    `json:"group"` // feed group/folder name, "" means ungrouped
	LastRefreshSuccess int       `json:"last_refresh_success"` // 新文章数，-1=失败
	LastRefreshError   string    `json:"last_refresh_error"`   // 失败错误信息
	LastRefreshed      time.Time `json:"last_refreshed"`       // 最后刷新时间
	UnreadCount        int       `json:"unread_count"`         // 未读文章数
}

type Article struct {
	ID           int64     `json:"id"`
	FeedID       int64     `json:"feed_id"`
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	Content      string    `json:"content"`
	Summary      string    `json:"summary"`
	Author       string    `json:"author"`
	Published    time.Time `json:"published"`
	IsFiltered   bool      `json:"is_filtered"`
	IsSaved      bool      `json:"is_saved"`
	Status       string    `json:"status"` // "unread", "accepted", "rejected", "snoozed"
	CreatedAt         time.Time `json:"created_at"`
	IsTranslated      bool      `json:"is_translated"`
	TranslatedContent string    `json:"translated_content"`
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
	Provider      string `json:"provider"`   // openai, claude, ollama
	APIKey        string `json:"api_key"`
	BaseURL       string `json:"base_url"`
	Model         string `json:"model"`
	MaxTokens     int    `json:"max_tokens"`
	ContextWindow int    `json:"context_window"` // 新增
	OutputReserve int    `json:"output_reserve"` // 新增
}

// AppState holds the application state for frontend
type AppState struct {
	Feeds         []Feed         `json:"feeds"`
	Articles      []Article      `json:"articles"`
	Notes         []Note         `json:"notes"`
	AIConfig      AIProviderConfig `json:"ai_config"`
	FilterMode    string         `json:"filter_mode"` // all, filtered, saved
}

// Briefing is an AI-generated daily briefing
type Briefing struct {
	ID          int64            `json:"id"`
	Status      string           `json:"status"` // pending, generating, completed, failed
	Title       string           `json:"title,omitempty"`
	Lead        string           `json:"lead,omitempty"`
	Closing     string           `json:"closing,omitempty"`
	Error       string           `json:"error,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Items       []BriefingItem   `json:"items,omitempty"`
}

// BriefingItem is a section within a briefing (formerly "topic")
type BriefingItem struct {
	ID         int64             `json:"id"`
	BriefingID int64             `json:"briefing_id"`
	Topic      string            `json:"topic"`  // section name, e.g. "AI领域" / "新能源与汽车"
	Summary    string            `json:"summary"` // section summary (1-2句)
	SortOrder  int               `json:"sort_order"`
	Articles   []BriefingArticle `json:"articles"`
}

// BriefingArticle is a reference to an article within a briefing item
type BriefingArticle struct {
	ID             int64  `json:"id"`
	BriefingItemID int64  `json:"briefing_item_id"`
	ArticleID      int64  `json:"article_id"`
	Title          string `json:"title"`
	Insight        string `json:"insight,omitempty"`
	KeyArgument    string `json:"key_argument,omitempty"` // 核心论点
	SourceURL      string `json:"source_url,omitempty"`   // 文章链接
}

// BriefingTopicArticle is the AI output format for an article within a section
type BriefingTopicArticle struct {
	ID          int64  `json:"id"`
	Insight     string `json:"insight"`
	KeyArgument string `json:"key_argument"` // 核心论点
	SourceURL   string `json:"source_url"`   // 文章链接
}

// BriefingTopic is the AI output format for a section
type BriefingTopic struct {
	Name     string                  `json:"name"`  // section name, e.g. "AI领域"
	Summary  string                  `json:"summary"` // section summary (1-2句)
	Articles []BriefingTopicArticle `json:"articles"`
}

// BriefingResult is the AI output format
type BriefingResult struct {
	Title    string           `json:"title"`    // kept for parsing but not stored
	Sections []BriefingTopic  `json:"sections"` // renamed from Topics
}

// PromptConfig holds prompt template configuration
type PromptConfig struct {
	ID        int64  `json:"id"`
	Type     string `json:"type"`      // summary, briefing, translation
	Name     string `json:"name"`      // display name
	Prompt   string `json:"prompt"`    // user prompt template
	System   string `json:"system"`    // system prompt
	MaxTokens int    `json:"max_tokens"`
	IsDefault bool   `json:"is_default"`
}
