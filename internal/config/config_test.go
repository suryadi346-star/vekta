package config

import (
    "os"
    "testing"
)

func TestLoad_ValidJSON(t *testing.T) {
    tmp, _ := os.CreateTemp("", "cfg*.json")
    tmp.WriteString(`{"system":{"ollama_host":"localhost:11434","memory_hard_limit_mb":512,"gogc":50},"cache":{"sqlite_path":"/tmp/x.db","ttl_minutes":30},"model":{"default":"qwen2.5:0.5b"}}`)
    tmp.Close()
    defer os.Remove(tmp.Name())

    cfg, err := Load(tmp.Name())
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if cfg.System.OllamaHost != "localhost:11434" {
        t.Errorf("wrong host: %s", cfg.System.OllamaHost)
    }
    if cfg.System.MemoryHardLimitMB != 512 {
        t.Errorf("wrong memory limit: %d", cfg.System.MemoryHardLimitMB)
    }
}

func TestLoad_MissingFile(t *testing.T) {
    _, err := Load("/nonexistent/path.json")
    if err == nil {
        t.Fatal("expected error for missing file")
    }
}

func TestLoad_InvalidJSON(t *testing.T) {
    tmp, _ := os.CreateTemp("", "cfg*.json")
    tmp.WriteString(`{invalid json}`)
    tmp.Close()
    defer os.Remove(tmp.Name())

    _, err := Load(tmp.Name())
    if err == nil {
        t.Fatal("expected error for invalid json")
    }
}

func TestLoad_Defaults(t *testing.T) {
    tmp, _ := os.CreateTemp("", "cfg*.json")
    tmp.WriteString(`{}`)
    tmp.Close()
    defer os.Remove(tmp.Name())

    cfg, err := Load(tmp.Name())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.System.OllamaHost != "127.0.0.1:11434" {
        t.Errorf("default host wrong: %s", cfg.System.OllamaHost)
    }
    if cfg.Cache.TTLMinutes != 45 {
        t.Errorf("default TTL wrong: %d", cfg.Cache.TTLMinutes)
    }
    if cfg.Model.Default != "qwen2.5:0.5b" {
        t.Errorf("default model wrong: %s", cfg.Model.Default)
    }
    if cfg.System.GOGC != 50 {
        t.Errorf("default GOGC wrong: %d", cfg.System.GOGC)
    }
}

func TestLoad_EnvOverride(t *testing.T) {
    tmp, _ := os.CreateTemp("", "cfg*.json")
    tmp.WriteString(`{"system":{"ollama_host":"localhost:11434"}}`)
    tmp.Close()
    defer os.Remove(tmp.Name())

    os.Setenv("OLLAMA_HOST", "192.168.1.10:11434")
    os.Setenv("MEMORY_LIMIT_MB", "1024")
    defer os.Unsetenv("OLLAMA_HOST")
    defer os.Unsetenv("MEMORY_LIMIT_MB")

    cfg, err := Load(tmp.Name())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if cfg.System.OllamaHost != "192.168.1.10:11434" {
        t.Errorf("env override failed: %s", cfg.System.OllamaHost)
    }
    if cfg.System.MemoryHardLimitMB != 1024 {
        t.Errorf("memory env override failed: %d", cfg.System.MemoryHardLimitMB)
    }
}
