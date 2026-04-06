package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/fetch"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSService struct {
	feedRepo    *sqlite.FeedRepository
	articleRepo *sqlite.ArticleRepository
	parser      *gofeed.Parser
	fetcher     *fetch.Fetcher
}

func NewRSSService() *RSSService {
	// Create HTTP client that skips TLS verification
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: httpTransport, Timeout: 30 * time.Second}
	parser := gofeed.NewParser()
	parser.Client = httpClient

	return &RSSService{
		feedRepo:    sqlite.NewFeedRepository(),
		articleRepo: sqlite.NewArticleRepository(),
		parser:      parser,
		fetcher:     fetch.NewFetcher(),
	}
}

func (s *RSSService) AddFeed(url string, title string) (*models.Feed, error) {
	// Just save the feed URL without validation - refresh will fetch actual data later
	feedTitle := title
	if feedTitle == "" {
		feedTitle = "Untitled Feed"
	}
	newFeed := &models.Feed{
		Title:     feedTitle,
		URL:       url,
		CreatedAt: time.Now(),
	}

	if err := s.feedRepo.Create(newFeed); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, errors.New("this feed has already been added")
		}
		return nil, fmt.Errorf("failed to save feed: %w", err)
	}

	return newFeed, nil
}

func (s *RSSService) fetchArticles(feed *models.Feed) (int, error) {
	articles, err := s.parser.ParseURL(feed.URL)
	if err != nil {
		return 0, err
	}

	newCount := 0
	for _, item := range articles.Items {
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}

		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}
		article := &models.Article{
			FeedID:     feed.ID,
			Title:      item.Title,
			Link:       item.Link,
			Content:    content,
			Summary:    item.Description,
			Author:     author,
			Published:  published,
			IsFiltered: false,
			IsSaved:    false,
			Status:     "unread",
			CreatedAt:  time.Now(),
		}

		// Check if article already exists
		exists, _ := s.articleRepo.LinkExists(article.Link)
		if !exists {
			if err := s.articleRepo.Create(article); err != nil {
				log.Printf("warning: failed to save article %s: %v", article.Title, err)
			} else {
				newCount++
			}
		}
	}

	feed.LastFetched = time.Now()
	s.feedRepo.Update(feed)

	return newCount, nil
}

func (s *RSSService) RefreshAllFeeds() error {
	return s.RefreshAllFeedsWithProgress(nil)
}

// RefreshAllFeedsWithProgress refreshes all feeds with optional progress callback.
// If onProgress is nil, behaves exactly like RefreshAllFeeds.
// Callback signature: onProgress(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string)
// newCount is -1 if there was an error, otherwise it's the number of new articles fetched.
// errMsg is empty string on success, or the error message on failure.
func (s *RSSService) RefreshAllFeedsWithProgress(onProgress func(idx, total int, feedTitle string, feedId int64, newCount int, errMsg string)) error {
	feeds, err := s.feedRepo.GetAll()
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		return nil
	}

	total := len(feeds)

	// Concurrent: max 5 parallel
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for i, feed := range feeds {
		wg.Add(1)
		go func(f models.Feed, idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			newCount, err := s.fetchArticles(&f)
			if err != nil {
				log.Printf("refresh feed %s error: %v", f.Title, err)
				if onProgress != nil {
					onProgress(idx+1, total, f.Title, f.ID, -1, err.Error())
				}
			} else {
				if onProgress != nil {
					onProgress(idx+1, total, f.Title, f.ID, newCount, "")
				}
			}
		}(feed, i)
	}

	wg.Wait()

	return nil
}

func (s *RSSService) refreshFeedWithRetry(feedID int64) error {
	const maxRetries = 3

	feed, err := s.feedRepo.GetByID(feedID)
	if err != nil {
		return err
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		_, err := s.fetchArticles(feed)
		if err == nil {
			return nil
		}
		lastErr = err

		// Irrecoverable: 404/410 → mark dead, stop retrying
		if isHTTPNotFound(err) {
			s.feedRepo.MarkDead(feedID)
			return fmt.Errorf("feed %d dead (404/410): %w", feedID, err)
		}

		// Exponential backoff: 1s, 2s, 4s
		if i < maxRetries-1 {
			time.Sleep(time.Duration(1<<uint(i)) * time.Second)
		}
	}
	return lastErr
}

// isHTTPNotFound returns true if err indicates a 404 or 410 HTTP response
func isHTTPNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "404") ||
		strings.Contains(s, "410") ||
		strings.Contains(s, "not found") ||
		strings.Contains(s, "Gone") ||
		strings.Contains(s, "StatusCode: 404") ||
		strings.Contains(s, "StatusCode: 410")
}

func (s *RSSService) RefreshFeed(feedID int64) error {
	return s.refreshFeedWithRetry(feedID)
}

func (s *RSSService) GetFeeds() ([]models.Feed, error) {
	return s.feedRepo.GetAll()
}

func (s *RSSService) GetDeadFeeds() ([]models.Feed, error) {
	return s.feedRepo.GetDeadFeeds()
}

func (s *RSSService) GetFeed(id int64) (*models.Feed, error) {
	return s.feedRepo.GetByID(id)
}

func (s *RSSService) UpdateFeed(feed *models.Feed) error {
	return s.feedRepo.Update(feed)
}

func (s *RSSService) DeleteFeed(id int64) error {
	return s.feedRepo.Delete(id)
}

func (s *RSSService) SetArticleStatus(id int64, status string) error {
	return s.articleRepo.SetStatus(id, status)
}

func (s *RSSService) MarkFeedAsRead(feedId int64) error {
	return s.articleRepo.SetFeedArticlesStatus(feedId, "accepted")
}

func (s *RSSService) MarkAllAsRead() error {
	return s.articleRepo.SetAllArticlesStatus("unread", "accepted")
}

func (s *RSSService) GetArticles(feedID int64, filterMode string, limit, offset int) ([]models.Article, error) {
	if feedID > 0 {
		return s.articleRepo.GetByFeedID(feedID, limit, offset)
	}
	return s.articleRepo.GetAll(filterMode, limit, offset)
}

func (s *RSSService) GetArticle(id int64) (*models.Article, error) {
	return s.articleRepo.GetByID(id)
}

func (s *RSSService) SearchArticles(query string, limit int) ([]models.Article, error) {
	return s.articleRepo.Search(query, limit)
}

// RefreshArticle fetches the full article content from the original URL
// and updates the article in the database. Returns the updated article.
func (s *RSSService) RefreshArticle(id int64) (*models.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if article.Link == "" {
		return article, nil
	}
	fullContent, err := s.fetcher.FetchFullContent(article.Link)
	if err != nil {
		return article, nil // return original on failure
	}
	article.Content = fullContent
	article.Summary = truncate(fullContent, 300)
	if err := s.articleRepo.Update(article); err != nil {
		return article, err
	}
	// Translate if English (after fetching full content)
	if article.Link != "" {
		if err := s.TranslateArticle(article); err != nil {
			log.Printf("warning: translation failed for article %d: %v", article.ID, err)
			// Don't fail the refresh if translation fails
		}
	}
	return article, nil
}

// TranslateArticle translates article content to Chinese if it's English.
// Returns nil if translation succeeded or was skipped, error otherwise.
func (s *RSSService) TranslateArticle(article *models.Article) error {
	// Skip if already translated
	if article.IsTranslated && article.TranslatedContent != "" {
		return nil
	}

	// Detect language
	if !isEnglish(article.Content) {
		return nil // Not English, skip translation
	}

	// Get translation prompt from DB
	promptRepo := sqlite.NewPromptRepository()
	promptConfig, err := promptRepo.GetByType("translation")
	if err != nil || promptConfig == nil || promptConfig.Prompt == "" {
		// Fallback: use default prompt
		provider := ai.GetProvider()
		translated, err := provider.GenerateSummaryWithPrompt(
			article.Content,
			"你是一位精通中英文互译的专业翻译官。必须仅输出中文译文，禁止任何额外话语。",
			"将以下英文文章翻译成中文。严格保留原始Markdown格式，专业术语使用业界通用中文表达，语言风格地道通顺。\n\n"+article.Content,
		)
		if err != nil {
			log.Printf("translation failed for article %d: %v", article.ID, err)
			return err
		}
		article.TranslatedContent = translated
		article.IsTranslated = true
	} else {
		// Use configured prompt
		provider := ai.GetProvider()
		translated, err := provider.GenerateSummaryWithPrompt(article.Content, promptConfig.System, promptConfig.Prompt)
		if err != nil {
			log.Printf("translation failed for article %d: %v", article.ID, err)
			return err
		}
		article.TranslatedContent = translated
		article.IsTranslated = true
	}

	// Update article in DB
	if err := s.articleRepo.Update(article); err != nil {
		log.Printf("failed to save translation for article %d: %v", article.ID, err)
		return err
	}
	return nil
}

// truncate returns the first n chars of s, stripping HTML tags.
func truncate(s string, n int) string {
	s = stripHTML(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// stripHTML removes HTML tags from s.
func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	// Simple tag strip — good enough for summaries
	idx := 0
	out := make([]byte, 0, len(s))
	for idx < len(s) {
		if s[idx] == '<' {
			// skip to next '>'
			for idx < len(s) && s[idx] != '>' {
				idx++
			}
			idx++ // skip '>'
			continue
		}
		out = append(out, s[idx])
		idx++
	}
	return string(out)
}

// isEnglish returns true if the content appears to be English based on ASCII ratio.
// Content with >50% ASCII characters is considered English.
func isEnglish(content string) bool {
	if len(content) < 100 {
		return false
	}
	asciiCount := 0
	for _, r := range content {
		if r < 128 {
			asciiCount++
		}
	}
	return float64(asciiCount)/float64(len(content)) > 0.5
}
