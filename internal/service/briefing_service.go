package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type BriefingService struct {
	briefingRepo   *sqlite.BriefingRepository
	articleRepo    *sqlite.ArticleRepository
	feedRepo       *sqlite.FeedRepository
	LastRefreshAt  time.Time // 最后刷新时间
	LastBriefingAt time.Time // 最后生成简报时间
}

func NewBriefingService() *BriefingService {
	return &BriefingService{
		briefingRepo: sqlite.NewBriefingRepository(),
		articleRepo:  sqlite.NewArticleRepository(),
		feedRepo:     sqlite.NewFeedRepository(),
	}
}

// GenerateBriefing creates a new briefing from recent articles
func (s *BriefingService) GenerateBriefing() (*models.Briefing, error) {
	// 0. Check if already generated this round
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
	provider := ai.GetProvider()
	prompt := s.buildPrompt(articlesInput)

	result, err := provider.GenerateBriefing(prompt)
	if err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", err.Error())
		return nil, fmt.Errorf("AI generation: %w", err)
	}

	// 5. Parse AI result
	var briefingResult models.BriefingResult
	if err := json.Unmarshal([]byte(result), &briefingResult); err != nil {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", "invalid AI response")
		return nil, fmt.Errorf("parse AI result: %w", err)
	}

	// 6. Store briefing items
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
		for _, articleID := range topic.ArticleIDs {
			title := ""
			for _, a := range articles {
				if a.ID == articleID {
					title = a.Title
					break
				}
			}
			ba := &models.BriefingArticle{
				BriefingItemID: item.ID,
				ArticleID:     articleID,
				Title:         title,
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
		summary := a.Summary
		if summary == "" {
			summary = a.Content
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
		}
		sb.WriteString(fmt.Sprintf("摘要: %s\n", summary))
		sb.WriteString("---\n")
	}
	return sb.String()
}

func (s *BriefingService) buildPrompt(articlesInput string) string {
	return fmt.Sprintf(`System: 你是一个内容策划助手。给定一组文章，你需要：
1. 将文章按主题分组（相似内容的文章分到同一组）
2. 为每个主题起一个简短的名字（如"AI"、"创业"、"科技"）
3. 为每个主题提取核心观点（用简洁的 bullets，每条不超过 20 字）

输出格式（严格按 JSON 格式，不要有其他内容）：
{
  "topics": [
    {
      "name": "主题名称",
      "article_ids": [101, 102],
      "summary": "• 核心观点1\n• 核心观点2\n• 核心观点3"
    }
  ]
}

规则：
- 每个简报最多 5 个主题
- 每个主题最多 5 篇核心文章
- 只包含真正有价值的文章，无关内容请忽略
- 主题按文章数量排序（多的在前）
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
