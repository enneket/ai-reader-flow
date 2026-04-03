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
	GenerateBriefing(prompt string) (string, error)
	FilterArticle(content string, rules []string) (bool, error)
	GetEmbedding(text string) ([]float32, error)
}

// EmbeddingProvider is the interface for embedding-only backends (used by FilterService)
type EmbeddingProvider interface {
	GetEmbedding(text string) ([]float32, error)
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
	systemPrompt := "You are a helpful assistant that summarizes articles. Provide a concise summary in 2-3 sentences."
	userPrompt := fmt.Sprintf("Summarize the following article:\n\n%s", content)

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

func (p *OpenAIProvider) FilterArticle(content string, rules []string) (bool, error) {
	systemPrompt := "You are a helpful assistant that filters articles based on user preferences. Answer only 'yes' or 'no'."
	userPrompt := fmt.Sprintf("Should I read this article? Consider these preferences:\n%s\n\nArticle:\n%s", strings.Join(rules, "\n"), content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens": 10,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					lower := strings.ToLower(strings.TrimSpace(content))
					return strings.HasPrefix(lower, "yes"), nil
				}
			}
		}
	}

	return false, fmt.Errorf("unexpected response format")
}

func (p *OpenAIProvider) GetEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model": "text-embedding-3-small",
		"input": text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("openai embed: %w: %s", err, string(body))
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("openai embed: no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

func (p *ClaudeProvider) GenerateSummary(content string) (string, error) {
	systemPrompt := "You are a helpful assistant that summarizes articles. Provide a concise summary in 2-3 sentences."
	userPrompt := fmt.Sprintf("Summarize the following article:\n\n%s", content)

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

func (p *ClaudeProvider) FilterArticle(content string, rules []string) (bool, error) {
	systemPrompt := "You are a helpful assistant that filters articles. Answer only 'yes' or 'no'."
	userPrompt := fmt.Sprintf("Should I read this article?\nPreferences:\n%s\n\nArticle:\n%s", strings.Join(rules, "\n"), content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
		"system": systemPrompt,
		"max_tokens": 10,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				lower := strings.ToLower(strings.TrimSpace(text))
				return strings.HasPrefix(lower, "yes"), nil
			}
		}
	}

	return false, fmt.Errorf("unexpected response format")
}

func (p *ClaudeProvider) GetEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model":       "claude-embedding-3",
		"inputs": []string{text},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("claude embed: %w: %s", err, string(body))
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("claude embed: no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

func (p *OllamaProvider) GenerateSummary(content string) (string, error) {
	systemPrompt := "You are a helpful assistant that summarizes articles. Provide a concise summary in 2-3 sentences."
	userPrompt := fmt.Sprintf("Summarize the following article:\n\n%s", content)

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

func (p *OllamaProvider) FilterArticle(content string, rules []string) (bool, error) {
	systemPrompt := "You are an article filter. Answer only 'yes' or 'no'."
	userPrompt := fmt.Sprintf("Should I read this article?\nPreferences:\n%s\n\nArticle:\n%s", strings.Join(rules, "\n"), content)

	reqBody := map[string]interface{}{
		"model": p.Model,
		"prompt": userPrompt,
		"system": systemPrompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}

	if response, ok := result["response"].(string); ok {
		lower := strings.ToLower(strings.TrimSpace(response))
		return strings.HasPrefix(lower, "yes"), nil
	}

	return false, fmt.Errorf("unexpected response format")
}

func (p *OllamaProvider) GetEmbedding(text string) ([]float32, error) {
	reqBody := map[string]interface{}{
		"model": p.Model,
		"input": text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.BaseURL+"/api/embed", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ollama embed: %w: %s", err, string(body))
	}

	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("ollama embed: no embedding returned")
	}

	return result.Embeddings[0], nil
}
