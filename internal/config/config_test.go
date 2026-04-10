package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFile(t *testing.T) {
	// Save original state
	origConfig := AppConfig_
	defer func() { AppConfig_ = origConfig }()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a config file in the default location ./data
	dataDir := "./data"
	// But since DATA_DIR env var isn't set and AppConfig_ is nil, getDataDir returns ./data
	if err := os.MkdirAll(filepath.Join(dataDir), 0755); err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	configContent := `[ai_provider]
provider = "claude"
api_key = "test-key"
base_url = "https://api.anthropic.com"
model = "claude-3"
max_tokens = 1000

[app]
data_dir = "/tmp/data"
port = 9090

[cron]
enabled = false
times = ["09:00"]
`
	configPath := filepath.Join(dataDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Reset AppConfig_ to nil
	AppConfig_ = nil

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.AIProvider.Provider != "claude" {
		t.Errorf("AIProvider.Provider = %q, want %q", cfg.AIProvider.Provider, "claude")
	}
	if cfg.AIProvider.APIKey != "test-key" {
		t.Errorf("AIProvider.APIKey = %q, want %q", cfg.AIProvider.APIKey, "test-key")
	}
	if cfg.AIProvider.Model != "claude-3" {
		t.Errorf("AIProvider.Model = %q, want %q", cfg.AIProvider.Model, "claude-3")
	}
	if cfg.AIProvider.MaxTokens != 1000 {
		t.Errorf("AIProvider.MaxTokens = %d, want %d", cfg.AIProvider.MaxTokens, 1000)
	}
	if cfg.App.Port != 9090 {
		t.Errorf("App.Port = %d, want %d", cfg.App.Port, 9090)
	}
	if cfg.Cron.Enabled {
		t.Errorf("Cron.Enabled = %v, want false", cfg.Cron.Enabled)
	}
	if len(cfg.Cron.Times) != 1 || cfg.Cron.Times[0] != "09:00" {
		t.Errorf("Cron.Times = %v, want %v", cfg.Cron.Times, []string{"09:00"})
	}

	// Cleanup
	os.RemoveAll(dataDir)
}

func TestLoadConfigDefaults(t *testing.T) {
	// Save original state
	origConfig := AppConfig_
	defer func() { AppConfig_ = origConfig }()

	// Remove any existing config
	dataDir := "./data"
	os.RemoveAll(dataDir)

	// Reset AppConfig_ to nil
	AppConfig_ = nil

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// Should get defaults
	if cfg.AIProvider.Provider != "openai" {
		t.Errorf("AIProvider.Provider = %q, want %q", cfg.AIProvider.Provider, "openai")
	}
	if cfg.AIProvider.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("AIProvider.BaseURL = %q, want %q", cfg.AIProvider.BaseURL, "https://api.openai.com/v1")
	}
	if cfg.AIProvider.Model != "gpt-3.5-turbo" {
		t.Errorf("AIProvider.Model = %q, want %q", cfg.AIProvider.Model, "gpt-3.5-turbo")
	}
	if cfg.AIProvider.MaxTokens != 500 {
		t.Errorf("AIProvider.MaxTokens = %d, want %d", cfg.AIProvider.MaxTokens, 500)
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want %d", cfg.App.Port, 8080)
	}
	if !cfg.Cron.Enabled {
		t.Errorf("Cron.Enabled = %v, want true", cfg.Cron.Enabled)
	}
	if len(cfg.Cron.Times) != 1 || cfg.Cron.Times[0] != "08:00" {
		t.Errorf("Cron.Times = %v, want %v", cfg.Cron.Times, []string{"08:00"})
	}
}

func TestSaveConfig(t *testing.T) {
	// Save original state
	origConfig := AppConfig_
	defer func() { AppConfig_ = origConfig }()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a config with the temp directory as data dir
	cfg := &Config{
		AIProvider: AIProviderConfig{
			Provider:  "ollama",
			APIKey:    "local-key",
			BaseURL:   "http://localhost:11434",
			Model:     "llama2",
			MaxTokens: 2000,
		},
		App: AppConfig{
			DataDir: tmpDir,
			Port:    3000,
		},
		Cron: CronConfig{
			Enabled: false,
			Times:   []string{"09:00"},
		},
	}

	AppConfig_ = cfg

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Verify file was written
	configPath := filepath.Join(tmpDir, "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file at %s: %v", configPath, err)
	}

	content := string(data)
	if !strings.Contains(content, `provider = 'ollama'`) {
		t.Errorf("expected provider = 'ollama' in config file, got:\n%s", content)
	}
	if !strings.Contains(content, `model = 'llama2'`) {
		t.Errorf("expected model = 'llama2' in config file, got:\n%s", content)
	}
	if !strings.Contains(content, `port = 3000`) {
		t.Errorf("expected port = 3000 in config file, got:\n%s", content)
	}
}

func TestSaveConfigThenLoad(t *testing.T) {
	// Save original state
	origConfig := AppConfig_
	defer func() { AppConfig_ = origConfig }()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create and save config
	cfg := &Config{
		AIProvider: AIProviderConfig{
			Provider:  "claude",
			APIKey:    "loaded-key",
			BaseURL:   "https://api.anthropic.com",
			Model:     "claude-3-sonnet",
			MaxTokens: 2000,
		},
		App: AppConfig{
			DataDir: tmpDir,
			Port:    8888,
		},
		Cron: CronConfig{
			Enabled: true,
			Times:   []string{"09:00"},
		},
	}

	AppConfig_ = cfg

	err = SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Now reset AppConfig_ and load
	AppConfig_ = nil

	// We need to make the config file be found at ./data/config.toml
	// because getDataDir returns ./data when AppConfig_ is nil
	dataDir := "./data"
	os.MkdirAll(dataDir, 0755)

	// Copy the config file to where LoadConfig will look
	srcPath := filepath.Join(tmpDir, "config.toml")
	dstPath := filepath.Join(dataDir, "config.toml")
	src, _ := os.Open(srcPath)
	dst, _ := os.Create(dstPath)
	io.Copy(dst, src)
	src.Close()
	dst.Close()

	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg2.AIProvider.Provider != "claude" {
		t.Errorf("AIProvider.Provider = %q, want %q", cfg2.AIProvider.Provider, "claude")
	}
	if cfg2.App.Port != 8888 {
		t.Errorf("App.Port = %d, want %d", cfg2.App.Port, 8888)
	}

	os.RemoveAll(dataDir)
}

func TestGetDataDirWithNilConfig(t *testing.T) {
	orig := AppConfig_
	AppConfig_ = nil
	defer func() { AppConfig_ = orig }()

	dir := getDataDir()
	if dir != "./data" {
		t.Errorf("getDataDir() = %q, want %q", dir, "./data")
	}
}

func TestGetDataDirWithConfig(t *testing.T) {
	orig := AppConfig_
	AppConfig_ = &Config{
		App: AppConfig{
			DataDir: "/custom/path",
		},
	}
	defer func() { AppConfig_ = orig }()

	dir := getDataDir()
	if dir != "/custom/path" {
		t.Errorf("getDataDir() = %q, want %q", dir, "/custom/path")
	}
}
