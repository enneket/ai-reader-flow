package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type SummaryService struct {
	articleRepo *sqlite.ArticleRepository
	promptRepo  *sqlite.PromptRepository
}

func NewSummaryService() *SummaryService {
	return &SummaryService{
		articleRepo: sqlite.NewArticleRepository(),
		promptRepo:  sqlite.NewPromptRepository(),
	}
}

func (s *SummaryService) GenerateSummary(article *models.Article) (string, error) {
	provider := ai.GetProvider()

	content := article.Content
	if len(content) > 10000 {
		content = content[:10000]
	}

	// Try to get configurable prompt first, fallback to default
	promptConfig, err := s.promptRepo.GetDefault("summary")
	if err == nil && promptConfig != nil && promptConfig.Prompt != "" {
		summary, err := provider.GenerateSummaryWithPrompt(content, promptConfig.System, promptConfig.Prompt)
		if err == nil {
			article.Summary = summary
			s.articleRepo.Update(article)
			return summary, nil
		}
		// If custom prompt fails, log and fallback to default
		log.Printf("Custom prompt failed, falling back to default: %v", err)
	}

	summary, err := provider.GenerateSummary(content)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	// Update article with summary
	article.Summary = summary
	s.articleRepo.Update(article)

	return summary, nil
}

func (s *SummaryService) GenerateSummaryForArticle(articleID int64) (string, error) {
	article, err := s.articleRepo.GetByID(articleID)
	if err != nil {
		return "", err
	}

	// Skip if summary already exists
	if article.Summary != "" {
		return article.Summary, nil
	}

	summary, err := s.GenerateSummary(article)
	if err != nil {
		// Retry once after 5s delay
		time.Sleep(5 * time.Second)
		article, err := s.articleRepo.GetByID(articleID)
		if err != nil {
			return "", err
		}
		if article.Summary != "" {
			return article.Summary, nil
		}
		summary, err = s.GenerateSummary(article)
		if err != nil {
			return "", err
		}
	}
	return summary, nil
}

func (s *SummaryService) BatchGenerateSummaries(articleIDs []int64, concurrency int) error {
	if len(articleIDs) == 0 {
		return nil
	}
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errs []error

	for _, id := range articleIDs {
		wg.Add(1)
		go func(articleID int64) {
			defer func() {
				<-sem
				wg.Done()
				if r := recover(); r != nil {
					log.Printf("Panic in BatchGenerateSummaries for article %d: %v\n", articleID, r)
				}
			}()

			_, err := s.GenerateSummaryForArticle(articleID)
			if err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
				log.Printf("Warning: failed to generate summary for article %d: %v\n", articleID, err)
			}
		}(id)
	}
	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *SummaryService) FormatSummaryForDisplay(summary string) string {
	// Clean up the summary text
	summary = strings.TrimSpace(summary)
	summary = strings.ReplaceAll(summary, "\r\n", "\n")
	return summary
}
