package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/config"
	"ai-rss-reader/internal/models"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetFeeds returns all feeds
func (a *App) GetFeeds() []models.Feed {
	feeds, err := rssService.GetFeeds()
	if err != nil {
		return []models.Feed{}
	}
	return feeds
}

// AddFeed adds a new RSS feed
func (a *App) AddFeed(url string) (*models.Feed, error) {
	feed, err := rssService.AddFeed(url)
	if err != nil {
		return nil, fmt.Errorf("failed to add feed: %w", err)
	}
	return feed, nil
}

// DeleteFeed deletes a feed
func (a *App) DeleteFeed(id int64) error {
	return rssService.DeleteFeed(id)
}

// AcceptArticle marks article as accepted
func (a *App) AcceptArticle(id int64) error {
	return rssService.SetArticleStatus(id, "accepted")
}

// RejectArticle marks article as rejected
func (a *App) RejectArticle(id int64) error {
	return rssService.SetArticleStatus(id, "rejected")
}

// SnoozeArticle marks article as snoozed
func (a *App) SnoozeArticle(id int64) error {
	return rssService.SetArticleStatus(id, "snoozed")
}

// GetDeadFeeds returns feeds that returned 404/410
func (a *App) GetDeadFeeds() []models.Feed {
	feeds, err := rssService.GetDeadFeeds()
	if err != nil {
		return []models.Feed{}
	}
	return feeds
}

// DeleteDeadFeed deletes a dead feed
func (a *App) DeleteDeadFeed(id int64) error {
	return rssService.DeleteFeed(id)
}

// RefreshFeed refreshes a single feed
func (a *App) RefreshFeed(id int64) error {
	return rssService.RefreshFeed(id)
}

// RefreshAllFeeds refreshes all feeds and auto-filters new articles
func (a *App) RefreshAllFeeds() error {
	if err := rssService.RefreshAllFeeds(); err != nil {
		return err
	}
	newArticleIDs, err := filterService.FilterAllArticlesNew()
	if err != nil {
		return err
	}
	// Launch background goroutine to generate summaries — does not block refresh
	go func() {
		summaryService.BatchGenerateSummaries(newArticleIDs, 5)
	}()
	return nil
}

// GetArticles returns articles, optionally filtered by feedID and filterMode
func (a *App) GetArticles(feedID int64, filterMode string) []models.Article {
	articles, err := rssService.GetArticles(feedID, filterMode)
	if err != nil {
		return []models.Article{}
	}
	return articles
}

// GetArticle returns a single article by ID
func (a *App) GetArticle(id int64) *models.Article {
	article, err := rssService.GetArticle(id)
	if err != nil {
		return nil
	}
	return article
}

// FilterArticle filters a single article using AI
func (a *App) FilterArticle(id int64) (bool, error) {
	article, err := rssService.GetArticle(id)
	if err != nil {
		return false, err
	}
	return filterService.FilterArticle(article)
}

// FilterAllArticles applies filters to all articles
func (a *App) FilterAllArticles() error {
	return filterService.FilterAllArticles()
}

// GenerateSummary generates a summary for an article
func (a *App) GenerateSummary(id int64) (string, error) {
	return summaryService.GenerateSummaryForArticle(id)
}

// GetFilterRules returns all filter rules
func (a *App) GetFilterRules() []models.FilterRule {
	rules, err := filterService.GetRules()
	if err != nil {
		return []models.FilterRule{}
	}
	return rules
}

// AddFilterRule adds a new filter rule
func (a *App) AddFilterRule(ruleType, value, action string) error {
	return filterService.AddRule(ruleType, value, action)
}

// DeleteFilterRule deletes a filter rule
func (a *App) DeleteFilterRule(id int64) error {
	return filterService.DeleteRule(id)
}

// GetNotes returns all notes
func (a *App) GetNotes() []models.Note {
	notes, err := noteService.GetNotes()
	if err != nil {
		return []models.Note{}
	}
	return notes
}

// CreateNote creates a note from an article
func (a *App) CreateNote(articleID int64, summary string) (*models.Note, error) {
	article, err := rssService.GetArticle(articleID)
	if err != nil {
		return nil, fmt.Errorf("article not found: %w", err)
	}
	return noteService.CreateNote(article, summary)
}

// ReadNote reads the content of a note
func (a *App) ReadNote(noteID int64) (string, error) {
	note, err := noteService.GetNoteByArticleID(noteID)
	if err != nil {
		return "", fmt.Errorf("note not found: %w", err)
	}
	content, err := noteService.ReadNote(note)
	if err != nil {
		return "", fmt.Errorf("failed to read note: %w", err)
	}
	return content, nil
}

// DeleteNote deletes a note
func (a *App) DeleteNote(noteID int64) error {
	return noteService.DeleteNote(noteID)
}

// GetAIConfig returns the AI provider configuration
func (a *App) GetAIConfig() models.AIProviderConfig {
	cfg := config.AppConfig_
	if cfg == nil {
		cfg, _ = config.LoadConfig()
	}
	return models.AIProviderConfig{
		Provider:  cfg.AIProvider.Provider,
		APIKey:    cfg.AIProvider.APIKey,
		BaseURL:   cfg.AIProvider.BaseURL,
		Model:     cfg.AIProvider.Model,
		MaxTokens: cfg.AIProvider.MaxTokens,
	}
}

// SaveAIConfig saves the AI provider configuration
func (a *App) SaveAIConfig(provider, apiKey, baseURL, model string, maxTokens int) error {
	cfg := config.AppConfig_
	if cfg == nil {
		cfg, _ = config.LoadConfig()
	}
	cfg.AIProvider = config.AIProviderConfig{
		Provider:  provider,
		APIKey:    apiKey,
		BaseURL:   baseURL,
		Model:     model,
		MaxTokens: maxTokens,
	}
	if err := config.SaveConfig(cfg); err != nil {
		return err
	}
	ai.InitProvider(cfg.AIProvider)
	return nil
}

// GetAppState returns the complete application state
func (a *App) GetAppState() models.AppState {
	feeds := a.GetFeeds()
	articles := a.GetArticles(0, "all")
	rules := a.GetFilterRules()
	notes := a.GetNotes()
	aiConfig := a.GetAIConfig()

	return models.AppState{
		Feeds:       feeds,
		Articles:    articles,
		FilterRules: rules,
		Notes:       notes,
		AIConfig:    aiConfig,
		FilterMode:  "all",
	}
}

// OpenExternal opens a URL in the default browser
func (a *App) OpenExternal(url string) error {
	// This would be implemented with runtime.BrowserOpenURL in Wails
	return nil
}

// getNotesDir returns the notes directory path
func getNotesDir() string {
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "data", "notes")
	os.MkdirAll(dir, 0755)
	return dir
}
