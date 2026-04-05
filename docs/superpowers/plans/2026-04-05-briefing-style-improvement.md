# Briefing Style Improvement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the briefing prompt to produce more natural, conversational output instead of formulaic reports.

**Architecture:** Modify the `buildPrompt` function in `briefing_service.go` to use few-shot examples and remove formulaic expression patterns. No structural changes to the service — only prompt content changes.

**Tech Stack:** Go, AI provider (OpenAI/Claude/Ollama compatible)

---

## Task 1: Rewrite buildPrompt function

**Files:**
- Modify: `internal/service/briefing_service.go:316-359` (the `buildPrompt` function)

- [ ] **Step 1: Read the current prompt**

Read `internal/service/briefing_service.go` lines 316-359 to see the current `buildPrompt` function.

- [ ] **Step 2: Replace the prompt template**

Replace the existing prompt with a new one that includes few-shot examples and prohibits formulaic expressions.

The new prompt should:
1. Start with "你是一个科技新闻爱好者，给朋友分享今天看到的有趣内容" instead of formal role description
2. Add explicit prohibitions: "核心发现", "相比", "对...有参考价值", "该主题聚焦"
3. Add few-shot examples showing the target style
4. Keep JSON output format requirement (AI still needs structured data)
5. Allow more flexibility in summary length

New prompt structure:
```
System: [角色设定 - 科技新闻爱好者风格]
[要求 - 口语化、禁止公式化]
[Few-shot example - 1-2 complete topic examples]
[输出格式说明 - JSON]

User: [文章列表]
```

- [ ] **Step 3: Test the change**

Run the Go tests to ensure no breakage:
```bash
cd /home/dabao/code/ai-reader-flow && go test ./internal/service/... -v
```

Expected: existing tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/service/briefing_service.go
git commit -m "feat(briefing): improve prompt style with few-shot examples

- Replace formal role description with conversational framing
- Add prohibitions against formulaic expressions
- Add few-shot examples showing natural style
- Keep JSON output format

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Verification

After implementation, generate a briefing and check:
1. Insights no longer use "核心发现是..." formula
2. Summaries read more naturally
3. Topics feel like friend sharing news, not writing reports

If results still show formulaic patterns, iterate on the prompt examples.
