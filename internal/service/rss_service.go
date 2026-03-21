package service

import (
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSService struct {
	feedRepo    *sqlite.FeedRepository
	articleRepo *sqlite.ArticleRepository
	parser      *gofeed.Parser
}

func NewRSSService() *RSSService {
	return &RSSService{
		feedRepo:    sqlite.NewFeedRepository(),
		articleRepo: sqlite.NewArticleRepository(),
		parser:      gofeed.NewParser(),
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
			CreatedAt:  time.Now(),
		}

		// Check if article already exists
		exists, _ := s.articleRepo.LinkExists(article.Link)
		if !exists {
			s.articleRepo.Create(article)
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

	for _, feed := range feeds {
		if err := s.RefreshFeed(feed.ID); err != nil {
			fmt.Printf("Warning: failed to refresh feed %s: %v\n", feed.Title, err)
		}
	}

	return nil
}

func (s *RSSService) RefreshFeed(feedID int64) error {
	feed, err := s.feedRepo.GetByID(feedID)
	if err != nil {
		return err
	}

	return s.fetchArticles(feed)
}

func (s *RSSService) GetFeeds() ([]models.Feed, error) {
	return s.feedRepo.GetAll()
}

func (s *RSSService) DeleteFeed(id int64) error {
	return s.feedRepo.Delete(id)
}

func (s *RSSService) GetArticles(feedID int64, filterMode string) ([]models.Article, error) {
	if feedID > 0 {
		return s.articleRepo.GetByFeedID(feedID)
	}
	return s.articleRepo.GetAll(filterMode)
}

func (s *RSSService) GetArticle(id int64) (*models.Article, error) {
	return s.articleRepo.GetByID(id)
}
