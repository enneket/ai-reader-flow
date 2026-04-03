# 简报质量改进设计

## 问题

当前简报 AI 输出是标题堆砌——`buildArticlesInput` 只喂文章标题 + 200字摘要，AI 没有足够内容写真正的分析。

## 解决方案

### 1. 喂更多内容给 AI

`buildArticlesInput` 截断从 200 字提升到 **2000 字**（Article.Content），足够 AI 理解文章核心又不会撑爆 token。

### 2. 新的 JSON 输出结构

```json
{
  "topics": [
    {
      "name": "主题名称",
      "summary": "深度分析：这段在讲什么、为什么重要、和同组其他文章的异同",
      "articles": [
        {
          "id": 101,
          "insight": "这篇文章的核心发现是 X，相比同组其他文章的特点是 Y"
        }
      ]
    }
  ]
}
```

### 3. 新的 System Prompt

明确要求：
- 对每篇文章输出"一句话核心发现 + 为什么重要"
- 组级别有深度总结（不是标题罗列）
- 只保留真正有价值的文章

## 数据模型变更

`BriefingItem.Articles[]` 从只存 `BriefingArticle{Title}` → 变成 `BriefingArticle{ArticleID, Title, Insight}`

## 前端变更

`BriefingDetail.tsx` 展示调整：
- 组 summary：完整展示深度分析段落
- 每篇文章：`insight` 替代纯标题
- 保留 article 列表但加 insight 文字

## 文件变更

- `internal/service/briefing_service.go` — buildArticlesInput + buildPrompt + BriefingResult 结构
- `internal/models/models.go` — BriefingArticle 新增 Insight 字段
- `frontend/src/components/BriefingDetail.tsx` — 展示 article insight
