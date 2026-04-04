package service

import (
	"ai-rss-reader/internal/models"
	"math"
)

// ArticleRepo defines the repository methods needed by FilterService.
type ArticleRepo interface {
	GetAll(filterMode string, limit, offset int) ([]models.Article, error)
	SetFiltered(id int64, filtered bool) error
}

type FilterService struct {
	articleRepo ArticleRepo
}

func NewFilterService() *FilterService {
	return &FilterService{}
}

// CosineSimilarity computes cosine similarity between two vectors.
// Returns 0 for nil/empty/mismatched-length vectors.
func CosineSimilarity(a, b []float32) float64 {
	if a == nil || b == nil || len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dotProd float64
	var normA float64
	var normB float64
	for i := range a {
		dotProd += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProd / (math.Sqrt(normA) * math.Sqrt(normB))
}

// QualityScore returns 0-40 based on title clarity + content length.
// Threshold: 30 (articles below are auto-hidden).
func (s *FilterService) QualityScore(article *models.Article) int {
	titleScore := scoreTitle(article.Title)
	lengthScore := scoreLength(article.Content)
	return titleScore + lengthScore
}

// scoreTitle returns 0-25 based on title characteristics.
func scoreTitle(title string) int {
	length := len(title)
	if length < 5 || length > 150 {
		return 0
	}
	if length < 10 {
		return 5
	}
	if length <= 100 {
		score := 20
		// bonus for numbers
		for _, c := range title {
			if c >= '0' && c <= '9' {
				score += 5
				break
			}
		}
		// bonus for uppercase letters (excluding first letter)
		for _, c := range title[1:] {
			if c >= 'A' && c <= 'Z' {
				score += 5
				break
			}
		}
		if score > 25 {
			score = 25
		}
		return score
	}
	// 100 < length <= 150
	return 25
}

// scoreLength returns 0-15 based on content length.
func scoreLength(content string) int {
	length := len(content)
	if length < 500 {
		return 0
	}
	if length < 1000 {
		return 5
	}
	if length < 2000 {
		return 10
	}
	return 15
}

