// Command server starts the WhatsApp food-tracker HTTP service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/cache"
	"github.com/harshsantoshi-tech/food-tracker/internal/calculation"
	"github.com/harshsantoshi-tech/food-tracker/internal/config"
	"github.com/harshsantoshi-tech/food-tracker/internal/conversation"
	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	apphttp "github.com/harshsantoshi-tech/food-tracker/internal/http"
	"github.com/harshsantoshi-tech/food-tracker/internal/llm"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
	"github.com/harshsantoshi-tech/food-tracker/internal/storage"
	"github.com/harshsantoshi-tech/food-tracker/internal/whatsapp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := storage.NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("connected to postgres", "host", cfg.Postgres.Host, "db", cfg.Postgres.DBName)

	redisClient, err := cache.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer redisClient.Close()
	logger.Info("connected to redis", "addr", cfg.Redis.Addr)

	// --- storage layer ---
	userRepo := storage.NewUserRepository(db)
	userResolver := storage.NewUserIDResolver(userRepo)
	foodLogRepo := storage.NewFoodLogRepository(db)

	// --- food understanding (Phase 3) ---
	llmClient := llm.NewClient(cfg.LLM)
	foodParser := food.NewParser(llmClient)

	// --- nutrition (Phase 4), wrapped with Redis caching ---
	usdaProvider := nutrition.NewUSDAProvider(cfg.Nutrition)
	nutritionCache := cache.NewRedisNutritionCache(redisClient)
	nutritionProvider := nutrition.NewCachingProvider(usdaProvider, nutritionCache)

	// --- calculation (Phase 5) ---
	calcEngine := calculation.NewEngine()

	// --- conversation state + orchestration (Phase 6) ---
	convCache := cache.NewRedisConversationCache(redisClient)
	convStore := conversation.NewRedisStore(convCache)
	convManager := conversation.NewManager(foodParser, nutritionProvider, calcEngine, convStore, foodLogRepo)

	// --- WhatsApp (Phase 2), now wired to the real pipeline ---
	dedup := cache.NewRedisDeduplicator(redisClient)
	waClient := whatsapp.NewClient(cfg.WhatsApp)
	pipelineHandler := whatsapp.NewPipelineHandler(logger, userResolver, convManager, waClient)
	waHandler := whatsapp.NewHandler(logger, cfg.WhatsApp, dedup, pipelineHandler)

	server := apphttp.NewServer(logger, db, redisClient, waHandler, cfg.RequestTimeout)

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		logger.Info("starting http server", "port", cfg.HTTPPort, "env", cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server shut down cleanly")
	return nil
}