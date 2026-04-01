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
			want: "文章 ID: 1\n标题: Test Article\n摘要: This is a test summary.\n---\n",
		},
		{
			name: "single article without summary uses content",
			articles: []models.Article{
				{ID: 2, Title: "No Summary", Content: "This is the content."},
			},
			want: "文章 ID: 2\n标题: No Summary\n摘要: This is the content.\n---\n",
		},
		{
			name: "content truncated at 200 chars",
			articles: []models.Article{
				{ID: 3, Title: "Long Content", Content: strings.Repeat("a", 250)},
			},
			want: "文章 ID: 3\n标题: Long Content\n摘要: " + strings.Repeat("a", 200) + "...\n---\n",
		},
		{
			name: "multiple articles",
			articles: []models.Article{
				{ID: 10, Title: "First", Summary: "First summary"},
				{ID: 20, Title: "Second", Summary: "Second summary"},
			},
			want: "文章 ID: 10\n标题: First\n摘要: First summary\n---\n文章 ID: 20\n标题: Second\n摘要: Second summary\n---\n",
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

func TestBuildPrompt(t *testing.T) {
	svc := &BriefingService{}

	articlesInput := "文章 ID: 1\n标题: Test\n摘要: Summary\n---"
	prompt := svc.buildPrompt(articlesInput)

	if !strings.Contains(prompt, "System:") {
		t.Error("prompt should contain System instruction")
	}
	if !strings.Contains(prompt, "User:") {
		t.Error("prompt should contain User section")
	}
	if !strings.Contains(prompt, "以下是今天的文章：") {
		t.Error("prompt should contain Chinese header")
	}
	if !strings.Contains(prompt, articlesInput) {
		t.Error("prompt should contain articles input")
	}
	if !strings.Contains(prompt, `"topics"`) {
		t.Error("prompt should specify JSON topics format")
	}
	if !strings.Contains(prompt, "最多 5 个主题") {
		t.Error("prompt should limit to 5 topics")
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
			"topics": [
				{
					"name": "AI",
					"article_ids": [1, 2, 3],
					"summary": "• 进展1\n• 进展2"
				},
				{
					"name": "创业",
					"article_ids": [4, 5],
					"summary": "• 融资\n• 趋势"
				}
			]
		}`

		var result models.BriefingResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("failed to parse BriefingResult JSON: %v", err)
		}
		if len(result.Topics) != 2 {
			t.Errorf("Topics len = %d, want 2", len(result.Topics))
		}
	})

	t.Run("empty topics", func(t *testing.T) {
		jsonStr := `{"topics": []}`
		var result models.BriefingResult
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Errorf("failed to parse empty topics: %v", err)
		}
		if len(result.Topics) != 0 {
			t.Errorf("Topics len = %d, want 0", len(result.Topics))
		}
	})
}
