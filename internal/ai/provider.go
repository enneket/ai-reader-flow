package ai

import (
	"ai-rss-reader/internal/config"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIServiceProvider defines the interface for AI backends
type AIServiceProvider interface {
	GenerateSummary(content string) (string, error)
	GenerateSummaryWithPrompt(content, systemPrompt, userPrompt string) (string, error)
	GenerateBriefing(prompt string) (string, error)
}

// OpenAIProvider implements AIServiceProvider using OpenAI API
type OpenAIProvider struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

// ClaudeProvider implements AIServiceProvider using Claude API
type ClaudeProvider struct {
	APIKey  string
	BaseURL string
	Model   string
}

// OllamaProvider implements AIServiceProvider using Ollama local API
type OllamaProvider struct {
	BaseURL string
	Model   string
}

var currentProvider AIServiceProvider

func InitProvider(cfg config.AIProviderConfig) {
	switch strings.ToLower(cfg.Provider) {
	case "claude":
		currentProvider = &ClaudeProvider{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}
	case "ollama":
		currentProvider = &OllamaProvider{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}
	default: // openai or custom - use OpenAI-compatible format with user-provided base_url
		currentProvider = &OpenAIProvider{
			APIKey:    cfg.APIKey,
			BaseURL:   cfg.BaseURL,
			Model:     cfg.Model,
			MaxTokens: cfg.MaxTokens,
		}
	}
}

func GetProvider() AIServiceProvider {
	if currentProvider == nil {
		InitProvider(config.AIProviderConfig{
			Provider:  "openai",
			BaseURL:   "https://api.openai.com/v1",
			Model:     "gpt-3.5-turbo",
			MaxTokens: 500,
		})
	}
	return currentProvider
}

func (p *OpenAIProvider) GenerateSummary(content string) (string, error) {
	systemPrompt := `你是一名资深内容分析师，擅长用最极简的语言精准捕捉文章灵魂。输出必须为中文、客观、单段长句（可用逗号、句号，禁止分段/换行），禁止任何列表符号（- * 1.等），禁止出现"这篇文章讲了/摘要如下"等前置废话。`
	userPrompt := fmt.Sprintf(`为提供的文本创作一份"快读摘要"，旨在让读者在30秒内掌握核心情报。

要求：
1) 极简主义：剔除背景铺垫、案例细节、营销话术及修饰性词汇，直奔主题。
2) 内容密度：必须包含核心主体、关键动作/事件、最终影响/结论。
3) 篇幅：严格控制在50-150字之间。

待摘要内容：
%s`, content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens": p.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check for OpenAI error response
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return "", fmt.Errorf("openai error: %s", msg)
		}
		return "", fmt.Errorf("openai error: %v", errObj)
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *OpenAIProvider) GenerateSummaryWithPrompt(content, systemPrompt, userPrompt string) (string, error) {
	// Replace {content} placeholder in user prompt
	userPrompt = strings.ReplaceAll(userPrompt, "{content}", content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens": p.MaxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check for OpenAI error response
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return "", fmt.Errorf("openai error: %s", msg)
		}
		return "", fmt.Errorf("openai error: %v", errObj)
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *OpenAIProvider) GenerateBriefing(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 16384,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Check for OpenAI error response
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return "", fmt.Errorf("openai error: %s", msg)
		}
		return "", fmt.Errorf("openai error: %v", errObj)
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *ClaudeProvider) GenerateSummary(content string) (string, error) {
	systemPrompt := `你是一名资深内容分析师，擅长用最极简的语言精准捕捉文章灵魂。输出必须为中文、客观、单段长句（可用逗号、句号，禁止分段/换行），禁止任何列表符号（- * 1.等），禁止出现"这篇文章讲了/摘要如下"等前置废话。`
	userPrompt := fmt.Sprintf(`为提供的文本创作一份"快读摘要"，旨在让读者在30秒内掌握核心情报。

要求：
1) 极简主义：剔除背景铺垫、案例细节、营销话术及修饰性词汇，直奔主题。
2) 内容密度：必须包含核心主体、关键动作/事件、最终影响/结论。
3) 篇幅：严格控制在50-150字之间。

待摘要内容：
%s`, content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
		"system": systemPrompt,
		"max_tokens": p.Model,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *ClaudeProvider) GenerateSummaryWithPrompt(content, systemPrompt, userPrompt string) (string, error) {
	// Replace {content} placeholder in user prompt
	userPrompt = strings.ReplaceAll(userPrompt, "{content}", content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
		"system": systemPrompt,
		"max_tokens": 4096,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *ClaudeProvider) GenerateBriefing(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"system": "你是一个内容策划助手。",
		"max_tokens": 16384,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *OllamaProvider) GenerateSummary(content string) (string, error) {
	systemPrompt := `你是一名资深内容分析师，擅长用最极简的语言精准捕捉文章灵魂。输出必须为中文、客观、单段长句（可用逗号、句号，禁止分段/换行），禁止任何列表符号（- * 1.等），禁止出现"这篇文章讲了/摘要如下"等前置废话。`
	userPrompt := fmt.Sprintf(`为提供的文本创作一份"快读摘要"，旨在让读者在30秒内掌握核心情报。

要求：
1) 极简主义：剔除背景铺垫、案例细节、营销话术及修饰性词汇，直奔主题。
2) 内容密度：必须包含核心主体、关键动作/事件、最终影响/结论。
3) 篇幅：严格控制在50-150字之间。

待摘要内容：
%s`, content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"prompt": userPrompt,
		"system": systemPrompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *OllamaProvider) GenerateSummaryWithPrompt(content, systemPrompt, userPrompt string) (string, error) {
	// Replace {content} placeholder in user prompt
	userPrompt = strings.ReplaceAll(userPrompt, "{content}", content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"prompt": userPrompt,
		"system": systemPrompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}

	return "", fmt.Errorf("unexpected response format")
}

func (p *OllamaProvider) GenerateBriefing(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model":      p.Model,
		"prompt":     prompt,
		"stream":     false,
		"max_tokens": 65536,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}

	return "", fmt.Errorf("unexpected response format")
}

