package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"vekta/internal/api"
	"vekta/internal/cache"
	"vekta/internal/chat"
	"vekta/internal/config"
	"vekta/internal/model"
	"vekta/pkg/logger"
)

func main() {
	// --- Load config ---
	cfg, err := config.Load("configs/app.json")
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// --- Init logger ---
	logger.Init(logger.LevelInfo, "text")

	// --- Enforce memory limit ---
	if cfg.System.MemoryHardLimitMB > 0 {
		limitBytes := int64(cfg.System.MemoryHardLimitMB) * 1024 * 1024
		debug.SetMemoryLimit(limitBytes)
		slog.Info("memory hard limit set", "mb", cfg.System.MemoryHardLimitMB)
	}

	// --- Tuning GC & GOMAXPROCS ---
	if cfg.System.GOGC > 0 {
		debug.SetGCPercent(cfg.System.GOGC)
	}
	runtime.GOMAXPROCS(1)

	// --- Init cache ---
	sqliteCache, err := cache.NewSQLiteCache(cfg.Cache.SQLitePath)
	if err != nil {
		slog.Error("failed to init cache", "err", err)
		os.Exit(1)
	}
	defer sqliteCache.Close()

	evictDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-evictDone:
				return
			case <-ticker.C:
				n, err := sqliteCache.Evict()
				if err != nil {
					slog.Warn("cache eviction error", "err", err)
				} else if n > 0 {
					slog.Info("cache evicted", "rows", n)
				}
			}
		}
	}()

	// --- Init session manager ---
	sessionMgr := chat.NewManager(
		time.Duration(cfg.Cache.TTLMinutes)*time.Minute,
		20,
	)

	// --- Init model client ---
	ollamaClient := model.NewOllamaClient(
		fmt.Sprintf("http://%s", cfg.System.OllamaHost),
	)

	// --- Init handler ---
	handler := &api.Handler{
		Model:        ollamaClient,
		Cache:        sqliteCache,
		Sessions:     sessionMgr,
		DefaultModel: cfg.Model.Default,
	}

	// --- Router ---
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/stream", handler.ChatStream)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","sessions":%d}`, sessionMgr.Count())
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("vekta started", "addr", srv.Addr, "model", cfg.Model.Default)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")
	close(evictDone)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "err", err)
	}
	slog.Info("vekta stopped")
}
