package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"strings"
)

type FilterService struct {
	ruleRepo    *sqlite.FilterRuleRepository
	articleRepo *sqlite.ArticleRepository
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
	articles, err := s.articleRepo.GetAll("all")
	if err != nil {
		return err
	}

	for _, article := range articles {
		shouldShow, err := s.FilterArticle(&article)
		if err != nil {
			continue
		}
		s.articleRepo.SetFiltered(article.ID, !shouldShow)
	}

	return nil
}
