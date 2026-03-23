package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gr8cally/knowledge_base_RAG/internal/config"
	httpserver "github.com/gr8cally/knowledge_base_RAG/internal/http"
	"github.com/gr8cally/knowledge_base_RAG/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := observability.NewLogger(cfg.AppEnv)

	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
		logger.Error("failed to create sqlite dir", "error", err)
		os.Exit(1)
	}

	if _, err := os.OpenFile(cfg.SQLitePath, os.O_CREATE|os.O_RDWR, 0o644); err != nil {
		logger.Error("failed to open sqlite file", "error", err)
		os.Exit(1)
	}

	router := httpserver.NewRouter(logger, cfg.SQLitePath)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
