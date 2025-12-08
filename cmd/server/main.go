package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/handler"
	"github.com/leaderboard-redis/internal/kafka"
	"github.com/leaderboard-redis/internal/postgres"
	"github.com/leaderboard-redis/internal/redis"
	"github.com/leaderboard-redis/internal/service"
	"github.com/leaderboard-redis/internal/websocket"
	"github.com/leaderboard-redis/internal/worker"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("failed to load config file, using defaults", "error", err)
		cfg = config.DefaultConfig()
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis
	logger.Info("connecting to Redis", "addr", cfg.Redis.Addr)
	redisService, err := redis.NewLeaderboardService(&cfg.Redis, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisService.Close()
	logger.Info("connected to Redis")

	// Initialize PostgreSQL
	logger.Info("connecting to PostgreSQL", "host", cfg.Postgres.Host, "database", cfg.Postgres.Database)
	postgresRepo, err := postgres.NewRepository(&cfg.Postgres, logger)
	if err != nil {
		logger.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer postgresRepo.Close()
	logger.Info("connected to PostgreSQL")

	// Run database migrations
	if err := postgresRepo.RunMigrations(ctx); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(logger)
	go wsHub.Run()
	logger.Info("WebSocket hub initialized")

	// Initialize services
	leaderboardService := service.NewLeaderboardService(
		redisService,
		postgresRepo,
		&cfg.Leaderboard,
		logger,
	)

	// Set the WebSocket hub on the service for broadcasting
	leaderboardService.SetHub(wsHub)

	// Initialize sync worker
	syncWorker := worker.NewSyncWorker(
		redisService,
		postgresRepo,
		&cfg.Sync,
		logger,
	)

	// Sync from database to Redis on startup (recovery)
	logger.Info("syncing leaderboards from database to Redis")
	if err := syncWorker.SyncAllFromDatabase(ctx); err != nil {
		logger.Warn("failed to sync from database on startup", "error", err)
	}

	// Start sync worker
	if cfg.Sync.Enabled {
		if err := syncWorker.Start(ctx); err != nil {
			logger.Error("failed to start sync worker", "error", err)
			os.Exit(1)
		}
	}

	// Initialize Kafka consumer for high-load score ingestion
	var kafkaConsumer *kafka.Consumer
	if cfg.Kafka.Enabled {
		logger.Info("initializing Kafka consumer",
			"brokers", cfg.Kafka.Brokers,
			"topic", cfg.Kafka.Topic,
		)
		var err error
		kafkaConsumer, err = kafka.NewConsumer(&cfg.Kafka, leaderboardService, logger)
		if err != nil {
			logger.Warn("failed to create Kafka consumer, continuing without Kafka", "error", err)
		} else {
			if err := kafkaConsumer.Start(); err != nil {
				logger.Warn("failed to start Kafka consumer, continuing without Kafka", "error", err)
				kafkaConsumer = nil
			} else {
				logger.Info("Kafka consumer started successfully")
			}
		}
	}

	// Initialize HTTP handler with WebSocket hub
	httpHandler := handler.NewHandler(leaderboardService, wsHub, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      httpHandler.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", "port", cfg.Server.Port)
		logger.Info("WebSocket endpoint available at /ws")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop WebSocket hub
	wsHub.Stop()

	// Stop Kafka consumer
	if kafkaConsumer != nil {
		if err := kafkaConsumer.Stop(); err != nil {
			logger.Error("failed to stop Kafka consumer", "error", err)
		}
	}

	// Stop sync worker
	if err := syncWorker.Stop(); err != nil {
		logger.Error("failed to stop sync worker", "error", err)
	}

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown server", "error", err)
	}

	logger.Info("server stopped")
}
