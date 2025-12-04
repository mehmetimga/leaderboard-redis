package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/postgres"
	"github.com/leaderboard-redis/internal/redis"
)

// SyncWorker handles periodic synchronization between Redis and PostgreSQL
type SyncWorker struct {
	redis      *redis.LeaderboardService
	postgres   *postgres.Repository
	config     *config.SyncConfig
	logger     *slog.Logger
	stopCh     chan struct{}
	doneCh     chan struct{}
	mu         sync.Mutex
	running    bool
}

// NewSyncWorker creates a new sync worker
func NewSyncWorker(
	redis *redis.LeaderboardService,
	postgres *postgres.Repository,
	cfg *config.SyncConfig,
	logger *slog.Logger,
) *SyncWorker {
	return &SyncWorker{
		redis:    redis,
		postgres: postgres,
		config:   cfg,
		logger:   logger,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the background sync process
func (w *SyncWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	w.logger.Info("sync worker started", "interval", w.config.Interval)

	go w.run(ctx)
	return nil
}

// Stop stops the background sync process
func (w *SyncWorker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	close(w.stopCh)
	<-w.doneCh

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()

	w.logger.Info("sync worker stopped")
	return nil
}

// run is the main worker loop
func (w *SyncWorker) run(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.syncAll(ctx)
		}
	}
}

// syncAll syncs all leaderboards from Redis to PostgreSQL
func (w *SyncWorker) syncAll(ctx context.Context) {
	w.logger.Info("starting sync cycle")
	startTime := time.Now()

	// Get all leaderboards from PostgreSQL
	leaderboards, err := w.postgres.ListLeaderboards(ctx)
	if err != nil {
		w.logger.Error("failed to list leaderboards for sync", "error", err)
		return
	}

	syncedCount := 0
	errorCount := 0

	for _, lb := range leaderboards {
		if err := w.SyncToDatabase(ctx, lb.ID); err != nil {
			w.logger.Error("failed to sync leaderboard",
				"leaderboard_id", lb.ID,
				"error", err,
			)
			errorCount++
		} else {
			syncedCount++
		}
	}

	duration := time.Since(startTime)
	w.logger.Info("sync cycle completed",
		"duration", duration,
		"synced", syncedCount,
		"errors", errorCount,
	)
}

// SyncToDatabase syncs a leaderboard from Redis to PostgreSQL
func (w *SyncWorker) SyncToDatabase(ctx context.Context, leaderboardID string) error {
	w.logger.Debug("syncing leaderboard to database", "leaderboard_id", leaderboardID)

	// Get all scores from Redis
	entries, err := w.redis.GetAllScores(ctx, leaderboardID)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		w.logger.Debug("no scores to sync", "leaderboard_id", leaderboardID)
		return nil
	}

	// Convert to map for batch upsert
	scores := make(map[string]int64, len(entries))
	for _, entry := range entries {
		scores[entry.PlayerID] = entry.Score
	}

	// Batch upsert to PostgreSQL
	// Process in batches to avoid overwhelming the database
	batchSize := w.config.BatchSize
	if batchSize == 0 {
		batchSize = 1000
	}

	batch := make(map[string]int64, batchSize)
	count := 0

	for playerID, score := range scores {
		batch[playerID] = score
		count++

		if count >= batchSize {
			if err := w.postgres.BatchUpsertScores(ctx, leaderboardID, batch); err != nil {
				return err
			}
			batch = make(map[string]int64, batchSize)
			count = 0
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := w.postgres.BatchUpsertScores(ctx, leaderboardID, batch); err != nil {
			return err
		}
	}

	w.logger.Debug("synced leaderboard to database",
		"leaderboard_id", leaderboardID,
		"player_count", len(entries),
	)

	return nil
}

// SyncFromDatabase syncs a leaderboard from PostgreSQL to Redis
// This is useful for recovery or initialization
func (w *SyncWorker) SyncFromDatabase(ctx context.Context, leaderboardID string) error {
	w.logger.Debug("syncing leaderboard from database", "leaderboard_id", leaderboardID)

	// Get all scores from PostgreSQL
	scores, err := w.postgres.GetAllScores(ctx, leaderboardID)
	if err != nil {
		return err
	}

	if len(scores) == 0 {
		w.logger.Debug("no scores to sync from database", "leaderboard_id", leaderboardID)
		return nil
	}

	// Batch set scores in Redis
	if err := w.redis.BatchSetScores(ctx, leaderboardID, scores); err != nil {
		return err
	}

	w.logger.Debug("synced leaderboard from database",
		"leaderboard_id", leaderboardID,
		"player_count", len(scores),
	)

	return nil
}

// SyncAllFromDatabase syncs all leaderboards from PostgreSQL to Redis
func (w *SyncWorker) SyncAllFromDatabase(ctx context.Context) error {
	w.logger.Info("syncing all leaderboards from database")

	leaderboards, err := w.postgres.ListLeaderboards(ctx)
	if err != nil {
		return err
	}

	for _, lb := range leaderboards {
		if err := w.SyncFromDatabase(ctx, lb.ID); err != nil {
			w.logger.Error("failed to sync leaderboard from database",
				"leaderboard_id", lb.ID,
				"error", err,
			)
			// Continue with other leaderboards
		}

		// Also sync metadata
		if err := w.redis.SetLeaderboardMeta(ctx, lb); err != nil {
			w.logger.Warn("failed to sync leaderboard metadata",
				"leaderboard_id", lb.ID,
				"error", err,
			)
		}
	}

	w.logger.Info("completed syncing all leaderboards from database", "count", len(leaderboards))
	return nil
}

// IsRunning returns whether the worker is currently running
func (w *SyncWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// RunOnce runs a single sync cycle (useful for manual triggers)
func (w *SyncWorker) RunOnce(ctx context.Context) {
	w.syncAll(ctx)
}

