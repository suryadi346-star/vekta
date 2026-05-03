package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	System SystemConfig `json:"system"`
	Cache  CacheConfig  `json:"cache"`
	Model  ModelConfig  `json:"model"`
}

type SystemConfig struct {
	GOGC              int    `json:"gogc"`
	MallocArenaMax    int    `json:"malloc_arena_max"`
	OllamaHost        string `json:"ollama_host"`
	MemoryHardLimitMB int    `json:"memory_hard_limit_mb"`
}

type CacheConfig struct {
	SQLitePath  string `json:"sqlite_path"`
	LRUMaxItems int    `json:"lru_max_items"`
	TTLMinutes  int    `json:"ttl_minutes"`
}

type ModelConfig struct {
	Default    string `json:"default"`
	MaxTokens  int    `json:"max_tokens"`
	TimeoutSec int    `json:"timeout_sec"`
}

// Load membaca JSON config dan apply env override.
// Env vars: OLLAMA_HOST, MEMORY_LIMIT_MB, SQLITE_PATH
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config json: %w", err)
	}

	// Env override
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		cfg.System.OllamaHost = host
	}
	if limitStr := os.Getenv("MEMORY_LIMIT_MB"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			cfg.System.MemoryHardLimitMB = v
		}
	}
	if sqlitePath := os.Getenv("SQLITE_PATH"); sqlitePath != "" {
		cfg.Cache.SQLitePath = sqlitePath
	}

	// Defaults
	if cfg.System.OllamaHost == "" {
		cfg.System.OllamaHost = "127.0.0.1:11434"
	}
	if cfg.Cache.SQLitePath == "" {
		cfg.Cache.SQLitePath = "./cache/chat.db"
	}
	if cfg.Cache.TTLMinutes <= 0 {
		cfg.Cache.TTLMinutes = 45
	}
	if cfg.Model.Default == "" {
		cfg.Model.Default = "qwen2.5:0.5b"
	}
	if cfg.Model.MaxTokens <= 0 {
		cfg.Model.MaxTokens = 2048
	}
	if cfg.System.GOGC <= 0 {
		cfg.System.GOGC = 50
	}

	return &cfg, nil
}
