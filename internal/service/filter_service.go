package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"fmt"
	"log"
	"math"
	"strings"

	"golang.org/x/sync/errgroup"
)

// ArticleRepo defines the repository methods needed by FilterService.
type ArticleRepo interface {
	GetAll(filterMode string, limit, offset int) ([]models.Article, error)
	GetUnreadWithoutEmbedding() ([]models.Article, error)
	SaveEmbedding(id int64, embedding []float32) error
	UpdateQualityScore(id int64, score int) error
	SetFiltered(id int64, filtered bool) error
}

type FilterService struct {
	ruleRepo    *sqlite.FilterRuleRepository
	articleRepo ArticleRepo
}

func NewFilterService() *FilterService {
	return &FilterService{
		ruleRepo:    sqlite.NewFilterRuleRepository(),
		articleRepo: sqlite.NewArticleRepository(),
	}
}

func (s *FilterService) AddRule(ruleType, value, action string) error {
	rule := &models.FilterRule{
		Type:      ruleType,
		Value:     value,
		Action:    action,
		Enabled:   true,
		CreatedAt: "",
	}
	return s.ruleRepo.Create(rule)
}

func (s *FilterService) GetRules() ([]models.FilterRule, error) {
	return s.ruleRepo.GetAll()
}

func (s *FilterService) UpdateRule(rule *models.FilterRule) error {
	return s.ruleRepo.Update(rule)
}

func (s *FilterService) DeleteRule(id int64) error {
	return s.ruleRepo.Delete(id)
}

func (s *FilterService) FilterArticle(article *models.Article) (bool, error) {
	rules, err := s.ruleRepo.GetEnabled()
	if err != nil {
		return true, nil // Default to show if rules can't be loaded
	}

	if len(rules) == 0 {
		return true, nil
	}

	for _, rule := range rules {
		passed, err := s.evaluateRule(article, &rule)
		if err != nil {
			continue
		}
		if rule.Action == "exclude" && !passed {
			return false, nil
		}
		if rule.Action == "include" && passed {
			return true, nil
		}
	}

	// If AI filtering is enabled, use AI to decide
	return s.filterWithAI(article, rules)
}

func (s *FilterService) evaluateRule(article *models.Article, rule *models.FilterRule) (bool, error) {
	switch rule.Type {
	case "keyword":
		return s.matchKeyword(article, rule.Value), nil
	case "source":
		return s.matchSource(article, rule.Value), nil
	default:
		return true, nil
	}
}

func (s *FilterService) matchKeyword(article *models.Article, keyword string) bool {
	lowerKeyword := strings.ToLower(keyword)
	text := strings.ToLower(article.Title + " " + article.Content)
	return strings.Contains(text, lowerKeyword)
}

func (s *FilterService) matchSource(article *models.Article, source string) bool {
	return strings.Contains(strings.ToLower(article.Author), strings.ToLower(source))
}

func (s *FilterService) filterWithAI(article *models.Article, rules []models.FilterRule) (bool, error) {
	// Build preference text from rules
	var preferences []string
	for _, rule := range rules {
		if rule.Type == "ai_preference" {
			preferences = append(preferences, rule.Value)
		}
	}

	if len(preferences) == 0 {
		return true, nil
	}

	provider := ai.GetProvider()
	return provider.FilterArticle(article.Title+"\n\n"+article.Content, preferences)
}

func (s *FilterService) FilterAllArticles() error {
	const batchSize = 100
	for offset := 0; ; offset += batchSize {
		articles, err := s.articleRepo.GetAll("all", batchSize, offset)
		if err != nil {
			return err
		}
		if len(articles) == 0 {
			break
		}

		for _, article := range articles {
			shouldShow, err := s.FilterArticle(&article)
			if err != nil {
				continue
			}
			s.articleRepo.SetFiltered(article.ID, !shouldShow)
		}
	}

	return nil
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

// semanticDedupBatch marks articles for filtering that are semantically duplicate
// within the same batch. Returns a map of articleID -> true if should be filtered.
// When two articles have cosine similarity > 0.85, the one with lower quality score
// is marked for filtering.
func (s *FilterService) semanticDedupBatch(articles []models.Article, embeddings map[int64][]float32) map[int64]bool {
	toFilter := make(map[int64]bool)

	for i := 0; i < len(articles); i++ {
		for j := i + 1; j < len(articles); j++ {
			a, b := &articles[i], &articles[j]
			embA, embB := embeddings[a.ID], embeddings[b.ID]
			if embA == nil || embB == nil {
				continue
			}

			sim := CosineSimilarity(embA, embB)
			if sim > 0.85 {
				scoreA := s.QualityScore(a)
				scoreB := s.QualityScore(b)
				// lower score gets filtered
				if scoreA >= scoreB {
					toFilter[b.ID] = true
				} else {
					toFilter[a.ID] = true
				}
			}
		}
	}
	return toFilter
}

// FilterAllArticlesNew is the new Plan B implementation that computes embeddings,
// deduplicates semantically within the batch, and scores quality.
// Returns the IDs of articles that passed filtering (new + not filtered).
func (s *FilterService) FilterAllArticlesNew() ([]int64, error) {
	newArticles, err := s.articleRepo.GetUnreadWithoutEmbedding()
	if err != nil {
		return nil, err
	}
	if len(newArticles) == 0 {
		return nil, nil
	}

	// Step 1: compute embeddings in parallel
	provider := ai.GetProvider()
	type result struct {
		id    int64
		emb   []float32
		err   error
	}
	results := make(chan result, len(newArticles))
	sem := make(chan struct{}, 10)
	var wg errgroup.Group

	for i := range newArticles {
		wg.Go(func() error {
			a := &newArticles[i]
			sem <- struct{}{}
			defer func() { <-sem }()

			text := a.Title + "\n\n" + a.Summary
			if len(text) > 50000 {
				text = text[:50000]
			}
			emb, err := provider.GetEmbedding(text)
			results <- result{id: a.ID, emb: emb, err: err}
			return nil
		})
	}
	wg.Wait()
	close(results)

	// collect errors and embeddings
	var errs []error
	embeddings := make(map[int64][]float32)
	for res := range results {
		if res.err != nil {
			errs = append(errs, fmt.Errorf("embedding article %d: %w", res.id, res.err))
		} else {
			embeddings[res.id] = res.emb
		}
	}
	if len(errs) > 0 && len(embeddings) == 0 {
		// All embeddings failed - log warning but don't fail the whole operation
		// Just return no new articles to process
		log.Printf("Warning: all embeddings failed, skipping filter: %v", errs)
		return nil, nil
	}
	if len(errs) > 0 {
		log.Printf("Warning: some embeddings failed (%d/%d), continuing with available ones",
			len(embeddings), len(newArticles))
	}

	// save embeddings to DB
	for id, emb := range embeddings {
		if err := s.articleRepo.SaveEmbedding(id, emb); err != nil {
			return nil, fmt.Errorf("save embedding %d: %w", id, err)
		}
	}

	// Step 2: semantic dedup within batch
	toFilter := s.semanticDedupBatch(newArticles, embeddings)

	// Step 3: quality score and mark filtered; collect passing IDs
	var passedIDs []int64
	for i := range newArticles {
		a := &newArticles[i]
		if toFilter[a.ID] {
			s.articleRepo.SetFiltered(a.ID, true)
			continue
		}
		score := s.QualityScore(a)
		s.articleRepo.UpdateQualityScore(a.ID, score)
		if score < 30 {
			s.articleRepo.SetFiltered(a.ID, true)
		} else {
			passedIDs = append(passedIDs, a.ID)
		}
	}

	return passedIDs, nil
}
