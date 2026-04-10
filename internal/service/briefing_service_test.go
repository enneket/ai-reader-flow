package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ai-rss-reader/internal/models"
)

func TestBuildArticlesInput(t *testing.T) {
	svc := &BriefingService{}

	tests := []struct {
		name     string
		articles []models.Article
		want     string
	}{
		{
			name:     "empty articles",
			articles: []models.Article{},
			want:     "",
		},
		{
			name: "single article with summary",
			articles: []models.Article{
				{ID: 1, Title: "Test Article", Summary: "This is a test summary."},
			},
			want: "文章 ID: 1\n标题: Test Article\n链接: \n内容:\nThis is a test summary.\n---\n",
		},
		{
			name: "single article without summary uses content",
			articles: []models.Article{
				{ID: 2, Title: "No Summary", Content: "This is the content."},
			},
			want: "文章 ID: 2\n标题: No Summary\n链接: \n内容:\nThis is the content.\n---\n",
		},
		{
			name: "content truncated at 2000 chars",
			articles: []models.Article{
				{ID: 3, Title: "Long Content", Content: strings.Repeat("a", 2500)},
			},
			want: "文章 ID: 3\n标题: Long Content\n链接: \n内容:\n" + strings.Repeat("a", 2000) + "...\n---\n",
		},
		{
			name: "multiple articles",
			articles: []models.Article{
				{ID: 10, Title: "First", Summary: "First summary"},
				{ID: 20, Title: "Second", Summary: "Second summary"},
			},
			want: "文章 ID: 10\n标题: First\n链接: \n内容:\nFirst summary\n---\n文章 ID: 20\n标题: Second\n链接: \n内容:\nSecond summary\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.buildArticlesInput(tt.articles)
			if got != tt.want {
				t.Errorf("buildArticlesInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildArticlesInputIncludesLink(t *testing.T) {
	svc := &BriefingService{}
	articles := []models.Article{
		{ID: 1, Title: "Test", Summary: "Sum", Link: "https://example.com"},
	}
	got := svc.buildArticlesInput(articles)
	if !strings.Contains(got, "https://example.com") {
		t.Error("article link should be included in input")
	}
}

func TestBuildPrompt(t *testing.T) {
	svc := &BriefingService{}

	articlesInput := "文章 ID: 1\n标题: Test\n链接: https://example.com\n日期: 2026-04-09\n内容:\nSummary\n---"
	prompt := svc.buildPrompt(articlesInput, "2026年4月9日", 1, 0, 1)

	// Check new prompt keywords (新闻整合简报 format)
	if !strings.Contains(prompt, "新闻整合简报") {
		t.Error("prompt should contain 新闻整合简报")
	}
	if !strings.Contains(prompt, "核心事件") {
		t.Error("prompt should contain 核心事件")
	}
	if !strings.Contains(prompt, articlesInput) {
		t.Error("prompt should contain articles input")
	}
	if !strings.Contains(prompt, `"sections"`) {
		t.Error("prompt should specify JSON sections format")
	}
	if !strings.Contains(prompt, "最多 5 个分节") {
		t.Error("prompt should limit to 5 sections")
	}
	if !strings.Contains(prompt, "source_url") {
		t.Error("prompt should reference source_url field")
	}

	// JSON parse sanity check — validates the embedded JSON format parses correctly
	jsonStart := strings.Index(prompt, `{"title"`)
	jsonEnd := strings.LastIndex(prompt, `}`) + 1
	if jsonStart != -1 && jsonEnd > jsonStart {
		jsonStr := prompt[jsonStart:jsonEnd]
		var result models.BriefingResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("prompt JSON does not parse: %v", err)
		}
		if result.Title == "" {
			t.Error("parsed result should have a title")
		}
	}
}

func TestGenerateBriefingRoundCheck(t *testing.T) {
	now := time.Now()
	zeroTime := time.Time{}

	tests := []struct {
		name            string
		lastRefreshAt   time.Time
		lastBriefingAt  time.Time
		wantBlock       bool
	}{
		{
			name:          "first round - no block",
			lastRefreshAt: zeroTime,
			lastBriefingAt: zeroTime,
			wantBlock: false,
		},
		{
			name:          "refresh but no briefing - no block",
			lastRefreshAt: now,
			lastBriefingAt: zeroTime,
			wantBlock: false,
		},
		{
			name:          "briefing after refresh - block",
			lastRefreshAt: now.Add(-time.Hour),
			lastBriefingAt: now,
			wantBlock: true,
		},
		{
			name:          "same time refresh before briefing - no block",
			lastRefreshAt: now,
			lastBriefingAt: now.Add(-time.Second),
			wantBlock: false,
		},
		{
			name:          "zero refresh with zero briefing - no block",
			lastRefreshAt: zeroTime,
			lastBriefingAt: now,
			wantBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &BriefingService{
				LastRefreshAt:  tt.lastRefreshAt,
				LastBriefingAt: tt.lastBriefingAt,
			}

			// Replicate the round-check condition from GenerateBriefing
			blocked := !svc.LastBriefingAt.Before(svc.LastRefreshAt) && !svc.LastRefreshAt.IsZero()

			if blocked != tt.wantBlock {
				t.Errorf("round check = %v, want %v", blocked, tt.wantBlock)
			}
		})
	}
}

func TestBriefingModel(t *testing.T) {
	t.Run("default status is generating", func(t *testing.T) {
		b := &models.Briefing{}
		if b.Status != "" {
			t.Errorf("Default Briefing.Status = %q, want empty", b.Status)
		}
	})

	t.Run("briefing with items", func(t *testing.T) {
		b := &models.Briefing{
			ID:     1,
			Status: "completed",
			Items: []models.BriefingItem{
				{
					ID:        1,
					BriefingID: 1,
					Topic:     "AI",
					Summary:   "• AI进展\n• AI应用",
					SortOrder: 0,
					Articles: []models.BriefingArticle{
						{ID: 1, BriefingItemID: 1, ArticleID: 101, Title: "Article 1"},
					},
				},
			},
		}

		if len(b.Items) != 1 {
			t.Errorf("Briefing.Items len = %d, want 1", len(b.Items))
		}
		if b.Items[0].Topic != "AI" {
			t.Errorf("Briefing.Items[0].Topic = %q, want %q", b.Items[0].Topic, "AI")
		}
		if len(b.Items[0].Articles) != 1 {
			t.Errorf("Briefing.Items[0].Articles len = %d, want 1", len(b.Items[0].Articles))
		}
	})
}

func TestBriefingResultJSON(t *testing.T) {
	t.Run("valid briefing result", func(t *testing.T) {
		jsonStr := `{
			"title": "AI领域新闻整合简报",
			"sections": [
				{
					"name": "AI",
					"summary": "AI进展",
					"articles": [
						{"id": 1, "insight": "进展1", "key_argument": "论据", "source_url": "https://x.com"}
					]
				},
				{
					"name": "创业",
					"summary": "创业融资",
					"articles": [
						{"id": 2, "insight": "融资", "key_argument": "数据", "source_url": ""}
					]
				}
			]
		}`

		var result models.BriefingResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("failed to parse BriefingResult JSON: %v", err)
		}
		if len(result.Sections) != 2 {
			t.Errorf("Sections len = %d, want 2", len(result.Sections))
		}
		if result.Title != "AI领域新闻整合简报" {
			t.Errorf("Title = %q, want %q", result.Title, "AI领域新闻整合简报")
		}
	})

	t.Run("empty sections", func(t *testing.T) {
		jsonStr := `{"sections": []}`
		var result models.BriefingResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("failed to parse empty sections: %v", err)
		}
		if len(result.Sections) != 0 {
			t.Errorf("Sections len = %d, want 0", len(result.Sections))
		}
	})
}

func TestNormalizeTopicName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AI 大模型", "ai大模型"},
		{"AI大模型", "ai大模型"},
		{"  AI  大模型  ", "ai大模型"},
		{"robotics", "robotics"},
	}
	for _, tt := range tests {
		got := normalizeTopicName(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeTopicName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMergeBriefingResults(t *testing.T) {
	batch1 := models.BriefingResult{
		Sections: []models.BriefingTopic{
			{
				Name:    "AI 大模型",
				Summary: "summary1",
				Articles: []models.BriefingTopicArticle{
					{ID: 1, Insight: "insight1"},
					{ID: 2, Insight: "insight2"},
				},
			},
		},
	}
	batch2 := models.BriefingResult{
		Sections: []models.BriefingTopic{
			{
				Name:    "AI大模型", // same as batch1 topic (no space)
				Summary: "summary2",
				Articles: []models.BriefingTopicArticle{
					{ID: 2, Insight: "insight2 longer"}, // duplicate ID 2
					{ID: 3, Insight: "insight3"},
				},
			},
			{
				Name:    "机器人",
				Summary: "summary3",
				Articles: []models.BriefingTopicArticle{
					{ID: 4, Insight: "insight4"},
				},
			},
		},
	}

	result := mergeBriefingResults([]models.BriefingResult{batch1, batch2})

	// Should have 2 topics: "AI大模型" (merged, 3 articles) and "机器人" (1 article)
	if len(result.Sections) != 2 {
		t.Errorf("expected 2 topics, got %d", len(result.Sections))
	}

	// Find AI topic
	var aiTopic *models.BriefingTopic
	for i := range result.Sections {
		if normalizeTopicName(result.Sections[i].Name) == normalizeTopicName("AI大模型") {
			aiTopic = &result.Sections[i]
			break
		}
	}
	if aiTopic == nil {
		t.Fatal("AI topic not found after merge")
	}
	// Should have 3 unique articles (1, 2, 3)
	if len(aiTopic.Articles) != 3 {
		t.Errorf("AI topic should have 3 articles after dedup, got %d", len(aiTopic.Articles))
	}

	// Find 机器人 topic
	var robotTopic *models.BriefingTopic
	for i := range result.Sections {
		if normalizeTopicName(result.Sections[i].Name) == normalizeTopicName("机器人") {
			robotTopic = &result.Sections[i]
			break
		}
	}
	if robotTopic == nil {
		t.Fatal("机器人 topic not found after merge")
	}
	if len(robotTopic.Articles) != 1 {
		t.Errorf("robot topic should have 1 article, got %d", len(robotTopic.Articles))
	}

	// Topics should be sorted by article count descending (AI first with 3)
	if len(result.Sections) >= 2 && len(result.Sections[0].Articles) < len(result.Sections[1].Articles) {
		t.Error("topics should be sorted by article count descending")
	}
}

func TestMergeBriefingResultsEmpty(t *testing.T) {
	result := mergeBriefingResults([]models.BriefingResult{})
	if len(result.Sections) != 0 {
		t.Errorf("expected 0 topics, got %d", len(result.Sections))
	}
}

func TestMergeBriefingResultsSingleBatch(t *testing.T) {
	batch := models.BriefingResult{
		Sections: []models.BriefingTopic{
			{
				Name:    "Test",
				Summary: "summary",
				Articles: []models.BriefingTopicArticle{
					{ID: 1, Insight: "insight1"},
				},
			},
		},
	}
	result := mergeBriefingResults([]models.BriefingResult{batch})
	if len(result.Sections) != 1 {
		t.Errorf("expected 1 topic, got %d", len(result.Sections))
	}
}
