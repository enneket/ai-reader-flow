package service

import (
	"ai-rss-reader/internal/fetch"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"errors"
	"fmt"
	"log"
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
	return &RSSService{
		feedRepo:    sqlite.NewFeedRepository(),
		articleRepo: sqlite.NewArticleRepository(),
		parser:      gofeed.NewParser(),
		fetcher:     fetch.NewFetcher(),
	}
}

func (s *RSSService) AddFeed(url string) (*models.Feed, error) {
	feed, err := s.parser.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	iconURL := ""
	if feed.Image != nil {
		iconURL = feed.Image.URL
	}

	newFeed := &models.Feed{
		Title:       feed.Title,
		URL:         url,
		Description: feed.Description,
		IconURL:     iconURL,
		LastFetched: time.Now(),
		IsDead:      false,
		CreatedAt:   time.Now(),
	}

	if err := s.feedRepo.Create(newFeed); err != nil {
		return nil, fmt.Errorf("failed to save feed: %w", err)
	}

	// Fetch and store articles
	if err := s.fetchArticles(newFeed); err != nil {
		fmt.Printf("Warning: failed to fetch articles for feed %s: %v\n", newFeed.Title, err)
	}

	return newFeed, nil
}

func (s *RSSService) fetchArticles(feed *models.Feed) error {
	articles, err := s.parser.ParseURL(feed.URL)
	if err != nil {
		return err
	}

	for _, item := range articles.Items {
		published := time.Now()
		if item.PublishedParsed != nil {
			published = *item.PublishedParsed
		}

		content := item.Content
		if content == "" {
			content = item.Description
		}

		article := &models.Article{
			FeedID:     feed.ID,
			Title:      item.Title,
			Link:       item.Link,
			Content:    content,
			Summary:    item.Description,
			Author:     item.Author.Name,
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
			}
		}
	}

	feed.LastFetched = time.Now()
	s.feedRepo.Update(feed)

	return nil
}

func (s *RSSService) RefreshAllFeeds() error {
	feeds, err := s.feedRepo.GetAll()
	if err != nil {
		return err
	}

	if len(feeds) == 0 {
		return nil
	}

	// Concurrent: max 5 parallel
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, feed := range feeds {
		wg.Add(1)
		go func(f models.Feed) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := s.refreshFeedWithRetry(f.ID); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("feed %s: %w", f.Title, err))
				mu.Unlock()
			}
		}(feed)
	}

	wg.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
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
		err := s.fetchArticles(feed)
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

func (s *RSSService) DeleteFeed(id int64) error {
	return s.feedRepo.Delete(id)
}

func (s *RSSService) SetArticleStatus(id int64, status string) error {
	return s.articleRepo.SetStatus(id, status)
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
	return article, nil
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
