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
	"github.com/harshsantoshi-tech/food-tracker/internal/config"
	apphttp "github.com/harshsantoshi-tech/food-tracker/internal/http"
	"github.com/harshsantoshi-tech/food-tracker/internal/storage"
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

	server := apphttp.NewServer(logger, db, redisClient, cfg.RequestTimeout)

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