package ai

import (
	"ai-rss-reader/internal/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderGenerateSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			return
		}
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if req["model"] != "gpt-4" {
			t.Errorf("expected model gpt-4, got %v", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "This is a test summary.",
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4",
		MaxTokens: 500,
	}

	summary, err := provider.GenerateSummary("This is test article content.")
	if err != nil {
		t.Fatalf("GenerateSummary() error = %v", err)
	}

	if summary != "This is a test summary." {
		t.Errorf("GenerateSummary() = %q, want %q", summary, "This is a test summary.")
	}
}

func TestOpenAIProviderFilterArticle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "yes",
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4",
		MaxTokens: 10,
	}

	passed, err := provider.FilterArticle("article content", []string{"keyword: golang"})
	if err != nil {
		t.Fatalf("FilterArticle() error = %v", err)
	}

	if !passed {
		t.Errorf("FilterArticle() = false, want true")
	}
}

func TestOpenAIProviderFilterArticleNo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "no",
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4",
		MaxTokens: 10,
	}

	passed, err := provider.FilterArticle("article content", []string{"keyword: python"})
	if err != nil {
		t.Fatalf("FilterArticle() error = %v", err)
	}

	if passed {
		t.Errorf("FilterArticle() = true, want false")
	}
}

func TestClaudeProviderGenerateSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{
				{"text": "Claude summary here."},
			},
		})
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3",
	}

	summary, err := provider.GenerateSummary("Article content for summarization.")
	if err != nil {
		t.Fatalf("GenerateSummary() error = %v", err)
	}

	if summary != "Claude summary here." {
		t.Errorf("GenerateSummary() = %q, want %q", summary, "Claude summary here.")
	}
}

func TestOllamaProviderGenerateSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "Ollama summary output.",
		})
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "llama2",
	}

	summary, err := provider.GenerateSummary("Article content.")
	if err != nil {
		t.Fatalf("GenerateSummary() error = %v", err)
	}

	if summary != "Ollama summary output." {
		t.Errorf("GenerateSummary() = %q, want %q", summary, "Ollama summary output.")
	}
}

func TestOllamaProviderFilterArticle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "yes this is relevant",
		})
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "llama2",
	}

	passed, err := provider.FilterArticle("article content", []string{"keyword: golang"})
	if err != nil {
		t.Fatalf("FilterArticle() error = %v", err)
	}

	if !passed {
		t.Errorf("FilterArticle() = false, want true")
	}
}

func TestInitProviderOpenAI(t *testing.T) {
	cfg := config.AIProviderConfig{
		Provider:  "openai",
		APIKey:    "test-key",
		BaseURL:   "https://api.openai.com/v1",
		Model:     "gpt-4",
		MaxTokens: 500,
	}

	InitProvider(cfg)
	provider := GetProvider()
	if provider == nil {
		t.Fatalf("GetProvider() returned nil")
	}

	// Verify it's an OpenAI provider by checking the model
	openaiProvider, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected *OpenAIProvider, got %T", provider)
	}
	if openaiProvider.Model != "gpt-4" {
		t.Errorf("OpenAIProvider.Model = %q, want %q", openaiProvider.Model, "gpt-4")
	}
}

func TestInitProviderClaude(t *testing.T) {
	cfg := config.AIProviderConfig{
		Provider:  "claude",
		APIKey:    "test-key",
		BaseURL:   "https://api.anthropic.com",
		Model:     "claude-3",
		MaxTokens: 1000,
	}

	InitProvider(cfg)
	provider := GetProvider()
	if provider == nil {
		t.Fatalf("GetProvider() returned nil")
	}

	claudeProvider, ok := provider.(*ClaudeProvider)
	if !ok {
		t.Fatalf("expected *ClaudeProvider, got %T", provider)
	}
	if claudeProvider.Model != "claude-3" {
		t.Errorf("ClaudeProvider.Model = %q, want %q", claudeProvider.Model, "claude-3")
	}
}

func TestInitProviderOllama(t *testing.T) {
	cfg := config.AIProviderConfig{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "llama2",
	}

	InitProvider(cfg)
	provider := GetProvider()
	if provider == nil {
		t.Fatalf("GetProvider() returned nil")
	}

	ollamaProvider, ok := provider.(*OllamaProvider)
	if !ok {
		t.Fatalf("expected *OllamaProvider, got %T", provider)
	}
	if ollamaProvider.Model != "llama2" {
		t.Errorf("OllamaProvider.Model = %q, want %q", ollamaProvider.Model, "llama2")
	}
}

func TestGetProviderSingleton(t *testing.T) {
	// Reset the global provider
	currentProvider = nil

	provider := GetProvider()
	if provider == nil {
		t.Fatalf("GetProvider() returned nil")
	}

	// Second call should return the same instance
	provider2 := GetProvider()
	if provider != provider2 {
		t.Fatalf("GetProvider() returned different instances")
	}
}

func TestGetProviderInitializesDefault(t *testing.T) {
	// Reset the global provider
	currentProvider = nil

	provider := GetProvider()
	if provider == nil {
		t.Fatalf("GetProvider() returned nil")
	}

	// Should default to OpenAI
	_, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatalf("expected default *OpenAIProvider, got %T", provider)
	}
}

func TestOpenAIProviderGenerateSummaryUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return malformed response - empty choices
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4",
		MaxTokens: 500,
	}

	_, err := provider.GenerateSummary("content")
	if err == nil {
		t.Errorf("GenerateSummary() expected error for unexpected response")
	}
}

func TestClaudeProviderFilterArticleUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return malformed response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "claude-3",
	}

	_, err := provider.FilterArticle("content", []string{"rule"})
	if err == nil {
		t.Errorf("FilterArticle() expected error for unexpected response")
	}
}

func TestOllamaProviderGenerateSummaryUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return no response field
		json.NewEncoder(w).Encode(map[string]interface{}{
			"other": "data",
		})
	}))
	defer server.Close()

	provider := &OllamaProvider{
		BaseURL: server.URL,
		Model:   "llama2",
	}

	_, err := provider.GenerateSummary("content")
	if err == nil {
		t.Errorf("GenerateSummary() expected error for unexpected response")
	}
}

func TestOpenAIProviderFilterArticleUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return malformed response
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4",
		MaxTokens: 10,
	}

	_, err := provider.FilterArticle("content", []string{"rule"})
	if err == nil {
		t.Errorf("FilterArticle() expected error for unexpected response")
	}
}
