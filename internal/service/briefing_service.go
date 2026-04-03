package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/config"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	DefaultContextWindow = 32768
	DefaultOutputReserve = 2048
	DefaultPromptOverhead = 500
)

type BriefingService struct {
	briefingRepo   *sqlite.BriefingRepository
	articleRepo    *sqlite.ArticleRepository
	feedRepo       *sqlite.FeedRepository
	LastRefreshAt  time.Time // 最后刷新时间
	LastBriefingAt time.Time // 最后生成简报时间
	aiConfig       *config.AIProviderConfig
}

func NewBriefingService(aiConfig *config.AIProviderConfig) *BriefingService {
	return &BriefingService{
		briefingRepo: sqlite.NewBriefingRepository(),
		articleRepo:  sqlite.NewArticleRepository(),
		feedRepo:     sqlite.NewFeedRepository(),
		aiConfig:     aiConfig,
	}
}

// computeBudget returns the max tokens available for article content per batch.
// budget = contextWindow * 0.6 - promptOverhead - outputReserve
func (s *BriefingService) computeBudget() int {
	cw := DefaultContextWindow
	or := DefaultOutputReserve
	if s.aiConfig != nil {
		if s.aiConfig.ContextWindow > 0 {
			cw = s.aiConfig.ContextWindow
		}
		if s.aiConfig.OutputReserve > 0 {
			or = s.aiConfig.OutputReserve
		}
	}
	return cw*6/10 - DefaultPromptOverhead - or
}

// buildArticleStringForEstimate builds the article string for token estimation.
func (s *BriefingService) buildArticleStringForEstimate(a models.Article) string {
	content := a.Content
	if content == "" {
		content = a.Summary
	}
	return fmt.Sprintf("文章 ID: %d\n标题: %s\n内容:\n%s\n---\n", a.ID, a.Title, content)
}

// splitIntoBatches splits articles into token-budgeted batches.
// Returns a slice of article slices, each within the token budget.
func (s *BriefingService) splitIntoBatches(articles []models.Article) [][]models.Article {
	budget := s.computeBudget()
	var batches [][]models.Article
	var currentBatch []models.Article
	currentTokens := 0

	for _, a := range articles {
		articleStr := s.buildArticleStringForEstimate(a)
		articleTokens := ai.Estimate(articleStr)

		if currentTokens+articleTokens > budget && len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentTokens = 0
		}

		currentBatch = append(currentBatch, a)
		currentTokens += articleTokens
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// GenerateBriefing creates a new briefing from recent articles
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
	return s.GenerateBriefingWithProgress(nil)
}

// GenerateBriefingWithProgress creates a briefing with optional progress callback.
// If onProgress is nil, behaves exactly like GenerateBriefing.
func (s *BriefingService) GenerateBriefingWithProgress(onProgress func(stage, detail string)) (*models.Briefing, error) {
	// 0. Check if already generated this round
	if onProgress != nil {
		onProgress("checking", "检查生成状态...")
	}
	if !s.LastBriefingAt.Before(s.LastRefreshAt) && !s.LastRefreshAt.IsZero() {
		return nil, fmt.Errorf("本轮已生成简报，请稍后再试")
	}

	// 1. Create briefing record
	briefing := &models.Briefing{
		Status: "generating",
	}
	if err := s.briefingRepo.Create(briefing); err != nil {
		return nil, fmt.Errorf("create briefing: %w", err)
	}

	// 2. Get articles after last refresh
	if onProgress != nil {
		onProgress("fetching", "正在获取文章...")
	}
	articles, err := s.articleRepo.GetArticlesAfter(s.LastRefreshAt)
	if err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
		return nil, fmt.Errorf("get articles: %w", err)
	}

	if len(articles) == 0 {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", "暂无新文章")
		return nil, fmt.Errorf("暂无新文章")
	}

	// 3. Build articles input for AI
	articlesInput := s.buildArticlesInput(articles)

	// 4. Call AI to generate topics
	if onProgress != nil {
		onProgress("analyzing", "正在分析文章主题...")
	}
	provider := ai.GetProvider()
	prompt := s.buildPrompt(articlesInput)

	result, err := provider.GenerateBriefing(prompt)
	if err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// 5. Parse AI result — try direct JSON first, then extract from markdown code block
	var briefingResult models.BriefingResult
	parseErr := json.Unmarshal([]byte(result), &briefingResult)
	if parseErr == nil {
		// Direct parse OK
	} else {
		// Try to extract JSON from markdown code block
		idx := strings.Index(result, "{")
		if idx != -1 {
			jsonStr := strings.TrimSpace(result[idx:])
			endIdx := strings.LastIndex(jsonStr, "}")
			if endIdx != -1 {
				jsonStr = jsonStr[:endIdx+1]
				parseErr = json.Unmarshal([]byte(jsonStr), &briefingResult)
			}
		}
		if parseErr != nil {
			errMsg := fmt.Sprintf("parse failed: %v | raw: %s", parseErr, result)
			s.briefingRepo.UpdateStatus(briefing.ID, "failed", errMsg)
			log.Printf("[briefing] parse failed, raw response: %s", result)
			return nil, fmt.Errorf("parse AI result: %w", parseErr)
		}
	}

	// 6. Store briefing items
	if onProgress != nil {
		onProgress("generating", "正在生成简报...")
	}
	for i, topic := range briefingResult.Topics {
		item := &models.BriefingItem{
			BriefingID: briefing.ID,
			Topic:      topic.Name,
			Summary:    topic.Summary,
			SortOrder:  i,
		}
		if err := s.briefingRepo.CreateItem(item); err != nil {
			log.Printf("Warning: failed to create briefing item: %v", err)
			continue
		}

		// Store article references
		for _, ta := range topic.Articles {
			title := ""
			for _, a := range articles {
				if a.ID == ta.ID {
					title = a.Title
					break
				}
			}
			ba := &models.BriefingArticle{
				BriefingItemID: item.ID,
				ArticleID:     ta.ID,
				Title:         title,
				Insight:       ta.Insight,
			}
			s.briefingRepo.CreateArticle(ba)
		}
	}

	// 7. Mark as completed
	s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")
	s.LastBriefingAt = time.Now()

	return briefing, nil
}

func (s *BriefingService) buildArticlesInput(articles []models.Article) string {
	var sb strings.Builder
	for _, a := range articles {
		sb.WriteString(fmt.Sprintf("文章 ID: %d\n", a.ID))
		sb.WriteString(fmt.Sprintf("标题: %s\n", a.Title))
		content := a.Content
		if content == "" {
			content = a.Summary
		}
		// Feed up to 2000 chars so AI has enough context to write real analysis
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		sb.WriteString(fmt.Sprintf("内容:\n%s\n", content))
		sb.WriteString("---\n")
	}
	return sb.String()
}

func (s *BriefingService) buildPrompt(articlesInput string) string {
	return fmt.Sprintf(`System: 你是一个专业的内容分析助手。给定一组文章，你需要：

1. 将文章按主题分组（相似内容的文章分到同一组）
2. 为每个主题起一个精准的名字（如"Claude 4 发布"、"RISC-V 市场动态"）
3. 对每篇文章写出真正有价值的 insight：一句话核心发现 + 为什么重要（最多 2 句话）
4. 对整个主题写一段深度分析：这段在讲什么整体趋势、为什么值得关注、和同组其他文章的异同

输出格式（严格按 JSON，不要有其他内容）：
{
  "topics": [
    {
      "name": "主题名称",
      "summary": "深度分析：这段在讲什么、为什么重要、和同组其他文章的异同（100-200字）",
      "articles": [
        {"id": 101, "insight": "核心发现是X，相比同类文章的特点是Y（1-2句话）"},
        {"id": 102, "insight": "核心发现是X，相比同类文章的特点是Y（1-2句话）"}
      ]
    }
  ]
}

规则：
- 每个简报最多 5 个主题
- 每个主题最多 5 篇核心文章
- 只包含真正有价值的文章，无关内容请忽略
- 主题按文章数量排序（多的在前）
- summary 要有深度，不是标题罗列，而是真正帮助读者快速了解这个领域
- 如果文章太少或无价值，返回空的 topics 数组

User: 以下是今天的文章：
%s`, articlesInput)
}

// GetBriefingWithItems returns a briefing with all its items and articles
func (s *BriefingService) GetBriefingWithItems(id int64) (*models.Briefing, error) {
	briefing, err := s.briefingRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	items, err := s.briefingRepo.GetItemsByBriefingID(id)
	if err != nil {
		return nil, err
	}

	for i := range items {
		articles, err := s.briefingRepo.GetArticlesByItemID(items[i].ID)
		if err != nil {
			continue
		}
		items[i].Articles = articles
	}

	briefing.Items = items
	return briefing, nil
}

// GetAllBriefings returns all briefings
func (s *BriefingService) GetAllBriefings(limit, offset int) ([]models.Briefing, error) {
	return s.briefingRepo.GetAll(limit, offset)
}

// DeleteBriefing deletes a briefing
func (s *BriefingService) DeleteBriefing(id int64) error {
	return s.briefingRepo.Delete(id)
}
