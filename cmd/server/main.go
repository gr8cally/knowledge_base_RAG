package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"knowledge_base_RAG/internal/app"
	"knowledge_base_RAG/internal/config"
	httpserver "knowledge_base_RAG/internal/http"
	"knowledge_base_RAG/internal/observability"
	sqlitestore "knowledge_base_RAG/internal/storage/sqlite"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	logger := observability.NewLogger(cfg.AppEnv)
	deps := app.NewDependencies(cfg, logger)
	kbService := deps.NewKnowledgeBaseService()

	if _, err := deps.NewLLM(); err != nil {
		logger.Error("failed to initialize llm client", "error", err)
		os.Exit(1)
	}

	if _, err := deps.NewEmbedder(); err != nil {
		logger.Error("failed to initialize embedder", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlitestore.Initialize(ctx, cfg.SQLitePath); err != nil {
		logger.Error("failed to initialize sqlite", "error", err)
		os.Exit(1)
	}

	router := httpserver.NewRouter(logger, cfg.SQLitePath, deps.CheckChroma, kbService)

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
