package service

import (
	"ai-rss-reader/internal/ai"
	"ai-rss-reader/internal/config"
	"ai-rss-reader/internal/models"
	"ai-rss-reader/internal/repository/sqlite"
	"encoding/json"
	"fmt"
	"log"
	"sort"
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
	promptRepo     *sqlite.PromptRepository
	LastRefreshAt  time.Time // 最后刷新时间
	LastBriefingAt time.Time // 最后生成简报时间
	aiConfig       *config.AIProviderConfig
}

func NewBriefingService(aiConfig *config.AIProviderConfig) *BriefingService {
	return &BriefingService{
		briefingRepo: sqlite.NewBriefingRepository(),
		articleRepo:  sqlite.NewArticleRepository(),
		feedRepo:     sqlite.NewFeedRepository(),
		promptRepo:   sqlite.NewPromptRepository(),
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

// normalizeTopicName normalizes a topic name for merge matching.
// Converts to lowercase and removes spaces.
func normalizeTopicName(name string) string {
	return strings.ToLower(strings.Replace(strings.Replace(name, " ", "", -1), "\t", "", -1))
}

// mergeBriefingResults merges multiple BriefingResult batches into one.
// Sections with the same normalized name are merged (articles deduplicated by ID).
// Sections are sorted by article count (descending).
// Title/Lead/Closing are taken from the first non-empty batch.
func mergeBriefingResults(batches []models.BriefingResult) models.BriefingResult {
	sectionMap := make(map[string]*models.BriefingTopic)
	var result models.BriefingResult

	for _, batch := range batches {
		// Capture title from first non-empty
		if result.Title == "" && batch.Title != "" {
			result.Title = batch.Title
		}

		for i := range batch.Sections {
			section := &batch.Sections[i]
			key := normalizeTopicName(section.Name)
			existing, ok := sectionMap[key]
			if !ok {
				cloned := *section
				sectionMap[key] = &cloned
				continue
			}
			// Merge: deduplicate articles by ID
			seen := make(map[int64]bool)
			for _, a := range existing.Articles {
				seen[a.ID] = true
			}
			for _, a := range section.Articles {
				if !seen[a.ID] {
					existing.Articles = append(existing.Articles, a)
					seen[a.ID] = true
				}
			}
		}
	}

	sections := make([]models.BriefingTopic, 0, len(sectionMap))
	for _, t := range sectionMap {
		sections = append(sections, *t)
	}

	sort.Slice(sections, func(i, j int) bool {
		return len(sections[i].Articles) > len(sections[j].Articles)
	})

	result.Sections = sections
	return result
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

	// 3. Split articles into token-budgeted batches
	batches := s.splitIntoBatches(articles)
	totalBatches := len(batches)

	// 3b. Date range — not displayed to user
	dateRange := ""

	// 4. Call AI per batch, collect results
	if onProgress != nil {
		onProgress("analyzing", "正在分析文章主题...")
	}
	provider := ai.GetProvider()
	var allResults []models.BriefingResult

	for i, batch := range batches {
		if totalBatches > 1 && onProgress != nil {
			onProgress("analyzing", fmt.Sprintf("正在分析第 %d/%d 批...", i+1, totalBatches))
		}
		articlesInput := s.buildArticlesInput(batch)
		prompt := s.buildPrompt(articlesInput, dateRange, len(articles), i, totalBatches)

		result, err := provider.GenerateBriefing(prompt)
		if err != nil {
			log.Printf("[briefing] batch %d/%d AI error: %v", i+1, totalBatches, err)
			continue // Skip failed batch, keep others
		}

		briefingResult := s.parseAIResult(result)
		if briefingResult != nil {
			allResults = append(allResults, *briefingResult)
		}
	}

	if len(allResults) == 0 {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", "所有批次均失败")
		return nil, fmt.Errorf("AI 生成失败")
	}

	// 5. Merge results from all batches
	mergedResult := mergeBriefingResults(allResults)

	if len(mergedResult.Sections) == 0 {
		s.briefingRepo.UpdateStatus(briefing.ID, "failed", "无有效简报内容")
		return nil, fmt.Errorf("无有效简报内容")
	}

	// 6. Store briefing sections (formerly "topics")
	if onProgress != nil {
		onProgress("generating", "正在生成简报...")
	}
	for i, section := range mergedResult.Sections {
		item := &models.BriefingItem{
			BriefingID: briefing.ID,
			Topic:      section.Name,
			Summary:    section.Summary,
			SortOrder:  i,
		}
		if err := s.briefingRepo.CreateItem(item); err != nil {
			log.Printf("Warning: failed to create briefing item: %v", err)
			continue
		}

		// Store article references
		for _, ta := range section.Articles {
			title := ""
			for _, a := range articles {
				if a.ID == ta.ID {
					title = a.Title
					break
				}
			}
			ba := &models.BriefingArticle{
				BriefingItemID: item.ID,
				ArticleID:      ta.ID,
				Title:          title,
				Insight:        ta.Insight,
				KeyArgument:   ta.KeyArgument,
				SourceURL:      ta.SourceURL,
			}
			s.briefingRepo.CreateArticle(ba)
		}
	}

	// 7. Mark as completed
	s.briefingRepo.UpdateStatus(briefing.ID, "completed", "")
	s.LastBriefingAt = time.Now()

	return briefing, nil
}

// parseAIResult parses AI response into BriefingResult.
// Returns nil if parsing fails.
func (s *BriefingService) parseAIResult(result string) *models.BriefingResult {
	var briefingResult models.BriefingResult
	parseErr := json.Unmarshal([]byte(result), &briefingResult)
	if parseErr == nil {
		return &briefingResult
	}

	// Try to extract JSON from markdown code block
	idx := strings.Index(result, "{")
	if idx == -1 {
		log.Printf("[briefing] parse failed: no JSON found in response")
		return nil
	}
	jsonStr := strings.TrimSpace(result[idx:])
	endIdx := strings.LastIndex(jsonStr, "}")
	if endIdx == -1 {
		log.Printf("[briefing] parse failed: no closing brace in response")
		return nil
	}
	jsonStr = jsonStr[:endIdx+1]

	if parseErr = json.Unmarshal([]byte(jsonStr), &briefingResult); parseErr != nil {
		log.Printf("[briefing] parse failed: %v | raw: %s", parseErr, result)
		return nil
	}
	return &briefingResult
}

func (s *BriefingService) buildArticlesInput(articles []models.Article) string {
	var sb strings.Builder
	for _, a := range articles {
		sb.WriteString(fmt.Sprintf("文章 ID: %d\n", a.ID))
		sb.WriteString(fmt.Sprintf("标题: %s\n", a.Title))
		sb.WriteString(fmt.Sprintf("链接: %s\n", a.Link))
		content := a.Content
		if content == "" {
			content = a.Summary
		}
		// Feed up to 2000 chars so AI has enough context
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		sb.WriteString(fmt.Sprintf("内容:\n%s\n", content))
		sb.WriteString("---\n")
	}
	return sb.String()
}

func (s *BriefingService) buildPrompt(articlesInput string, dateRange string, totalArticles, batchIndex, totalBatches int) string {
	// Try to get configurable prompt first
	if s.promptRepo != nil {
		promptConfig, err := s.promptRepo.GetDefault("briefing")
		if err == nil && promptConfig != nil && promptConfig.Prompt != "" {
			// Replace placeholders in the template
			prompt := promptConfig.Prompt
			prompt = strings.ReplaceAll(prompt, "{articles}", articlesInput)
			prompt = strings.ReplaceAll(prompt, "{content}", articlesInput)
			prompt = strings.ReplaceAll(prompt, "{totalArticles}", fmt.Sprintf("%d", totalArticles))
			prompt = strings.ReplaceAll(prompt, "{batchIndex}", fmt.Sprintf("%d", batchIndex+1))
			prompt = strings.ReplaceAll(prompt, "{totalBatches}", fmt.Sprintf("%d", totalBatches))
			prompt = strings.ReplaceAll(prompt, "{topicLimit}", fmt.Sprintf("%d", 5))
			prompt = strings.ReplaceAll(prompt, "{dateRange}", dateRange)
			// Default length guidance if {length} not in template
			prompt = strings.ReplaceAll(prompt, "{length}", "500-800 字")
			return promptConfig.System + "\n" + prompt
		}
	}

	// Fallback to default prompt — 新闻整合简报格式
	isSingleBatch := totalBatches == 1
	sectionLimit := 5
	if !isSingleBatch {
		sectionLimit = 3
	}

	multiBatchNote := ""
	if !isSingleBatch {
		multiBatchNote = fmt.Sprintf("\n提示：这是第 %d/%d 批，请只关注本批内容。", batchIndex+1, totalBatches)
	}

	// JSON format for all batches — sections only
	headerPart := `【输出格式】（严格 JSON，不要有其他内容）
{
  "sections": [
    {
      "name": "分节名称，如"AI领域"",
      "summary": "本节要目（1-2句），如"涵盖模型进展与行业争议"",
      "articles": [
        {
          "id": 101,
          "insight": "一句话核心事件（时间+主体+核心事件+关键结果）",
          "key_argument": "关键结果或影响（1-2句）",
          "source_url": "https://..."
        }
      ]
    }
  ]
}

`
	return fmt.Sprintf(`根据以下文章，生成一份新闻整合简报。

【核心任务】
1. 阅读每篇文章，提炼核心事件（时间+主体+事件+结果）
2. 将文章按领域/主题分为若干分节（最多 %d 个分节）
3. 每条只保留核心事实，不添加主观评论

【输出格式】
%s【规则】
- 每节约 2-5 条新闻，新闻太少则合并到其他节
- 分节按新闻条数排序（多的在前）
- 只包含真正有价值的新闻，无关内容请忽略
- 不要在输出中包含任何日期（年、月、日），也不要用日期作为小标题%s

以下是文章（共 %d 篇，第 %d/%d 批）：
%s`, sectionLimit, headerPart, multiBatchNote, totalArticles, batchIndex+1, totalBatches, articlesInput)
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

// DeleteAllBriefings deletes all briefings
func (s *BriefingService) DeleteAllBriefings() error {
	return s.briefingRepo.DeleteAll()
}

// TryAcquireCronLock atomically acquires a cron lock for the given time slot.
func (s *BriefingService) TryAcquireCronLock(timeSlot string) bool {
	return s.briefingRepo.TryAcquireCronLock(timeSlot)
}

// ReleaseCronLock releases the cron lock for the given time slot.
func (s *BriefingService) ReleaseCronLock(timeSlot string) {
	s.briefingRepo.ReleaseCronLock(timeSlot)
}
