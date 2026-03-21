package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"fmt"
	"strings"
)

type SummaryService struct {
	articleRepo *sqlite.ArticleRepository
}

func NewSummaryService() *SummaryService {
	return &SummaryService{
		articleRepo: sqlite.NewArticleRepository(),
	}
}

func (s *SummaryService) GenerateSummary(article *models.Article) (string, error) {
	provider := ai.GetProvider()

	content := article.Content
	if len(content) > 10000 {
		content = content[:10000]
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

	return s.GenerateSummary(article)
}

func (s *SummaryService) BatchGenerateSummaries(articleIDs []int64) error {
	for _, id := range articleIDs {
		_, err := s.GenerateSummaryForArticle(id)
		if err != nil {
			fmt.Printf("Warning: failed to generate summary for article %d: %v\n", id, err)
		}
	}
	return nil
}

func (s *SummaryService) FormatSummaryForDisplay(summary string) string {
	// Clean up the summary text
	summary = strings.TrimSpace(summary)
	summary = strings.ReplaceAll(summary, "\r\n", "\n")
	return summary
}
