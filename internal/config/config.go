package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	AIProvider AIProviderConfig `toml:"ai_provider"`
	App        AppConfig        `toml:"app"`
	Cron       CronConfig       `toml:"cron"`
}

type AIProviderConfig struct {
	Provider  string `toml:"provider"`
	APIKey    string `toml:"api_key"`
	BaseURL   string `toml:"base_url"`
	Model     string `toml:"model"`
	MaxTokens int    `toml:"max_tokens"`
}

type AppConfig struct {
	DataDir string `toml:"data_dir"`
	Port    int    `toml:"port"`
}

type CronConfig struct {
	Enabled     bool `toml:"enabled"`
	IntervalMins int `toml:"interval_mins"`
	Hour        int  `toml:"hour"`        // 每日触发小时 (0-23)
	Minute      int  `toml:"minute"`      // 每日触发分钟 (0-59)
}

var AppConfig_ *Config

func LoadConfig() (*Config, error) {
	cfg := &Config{
		AIProvider: AIProviderConfig{
			Provider:  "openai",
			BaseURL:   "https://api.openai.com/v1",
			Model:     "gpt-3.5-turbo",
			MaxTokens: 500,
		},
		App: AppConfig{
			DataDir: "./data",
			Port:    8080,
		},
		Cron: CronConfig{
			Enabled:     true,
			IntervalMins: 30,
			Hour:        9,  // 默认早上 9 点
			Minute:      0,
		},
	}

	configPath := filepath.Join(getDataDir(), "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			toml.Unmarshal(data, cfg)
		}
	}

	AppConfig_ = cfg
	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	configPath := filepath.Join(getDataDir(), "config.toml")
	return os.WriteFile(configPath, data, 0644)
}

func getDataDir() string {
	if AppConfig_ != nil && AppConfig_.App.DataDir != "" {
		return AppConfig_.App.DataDir
	}
	return "./data"
}
