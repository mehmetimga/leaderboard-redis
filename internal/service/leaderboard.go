package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/domain"
	"github.com/leaderboard-redis/internal/postgres"
	"github.com/leaderboard-redis/internal/redis"
)

// LeaderboardService provides business logic for leaderboard operations
type LeaderboardService struct {
	redis    *redis.LeaderboardService
	postgres *postgres.Repository
	config   *config.LeaderboardConfig
	logger   *slog.Logger
}

// NewLeaderboardService creates a new leaderboard service
func NewLeaderboardService(
	redis *redis.LeaderboardService,
	postgres *postgres.Repository,
	cfg *config.LeaderboardConfig,
	logger *slog.Logger,
) *LeaderboardService {
	return &LeaderboardService{
		redis:    redis,
		postgres: postgres,
		config:   cfg,
		logger:   logger,
	}
}

// SubmitScore submits a score for a player
func (s *LeaderboardService) SubmitScore(ctx context.Context, submission domain.ScoreSubmission) error {
	// Get leaderboard config
	lbConfig, err := s.postgres.GetLeaderboard(ctx, submission.LeaderboardID)
	if err != nil {
		return fmt.Errorf("getting leaderboard config: %w", err)
	}

	// Apply score based on update mode
	switch lbConfig.UpdateMode {
	case domain.UpdateModeReplace:
		if err := s.redis.SetScore(ctx, submission.LeaderboardID, submission.PlayerID, submission.Score); err != nil {
			return fmt.Errorf("setting score in redis: %w", err)
		}
	case domain.UpdateModeIncrement:
		if _, err := s.redis.IncrementScore(ctx, submission.LeaderboardID, submission.PlayerID, submission.Score); err != nil {
			return fmt.Errorf("incrementing score in redis: %w", err)
		}
	case domain.UpdateModeBest:
		higherIsBetter := lbConfig.SortOrder == domain.SortOrderDesc
		if _, err := s.redis.SetScoreIfBetter(ctx, submission.LeaderboardID, submission.PlayerID, submission.Score, higherIsBetter); err != nil {
			return fmt.Errorf("setting best score in redis: %w", err)
		}
	default:
		if err := s.redis.SetScore(ctx, submission.LeaderboardID, submission.PlayerID, submission.Score); err != nil {
			return fmt.Errorf("setting score in redis: %w", err)
		}
	}

	// Record the event in PostgreSQL
	event := domain.ScoreEvent{
		PlayerID:      submission.PlayerID,
		LeaderboardID: submission.LeaderboardID,
		Score:         submission.Score,
		GameID:        submission.GameID,
		EventType:     "submit",
		Timestamp:     time.Now(),
		Metadata:      submission.Metadata,
	}
	if err := s.postgres.RecordEvent(ctx, event); err != nil {
		s.logger.Warn("failed to record score event", "error", err)
		// Don't fail the request if event recording fails
	}

	return nil
}

// SubmitScoreBatch submits multiple scores
func (s *LeaderboardService) SubmitScoreBatch(ctx context.Context, batch domain.BatchScoreSubmission) error {
	for _, submission := range batch.Scores {
		if err := s.SubmitScore(ctx, submission); err != nil {
			s.logger.Error("failed to submit score in batch",
				"player_id", submission.PlayerID,
				"leaderboard_id", submission.LeaderboardID,
				"error", err,
			)
			// Continue processing other scores
		}
	}
	return nil
}

// GetTopN returns the top N players from a leaderboard
func (s *LeaderboardService) GetTopN(ctx context.Context, leaderboardID string, n int) ([]domain.LeaderboardEntry, error) {
	// Validate limit
	if n <= 0 {
		n = s.config.DefaultLimit
	}
	if n > s.config.MaxLimit {
		n = s.config.MaxLimit
	}

	entries, err := s.redis.GetTopN(ctx, leaderboardID, n)
	if err != nil {
		return nil, fmt.Errorf("getting top n from redis: %w", err)
	}

	return entries, nil
}

// GetPlayerRank returns a player's rank and score
func (s *LeaderboardService) GetPlayerRank(ctx context.Context, leaderboardID, playerID string) (*domain.LeaderboardEntry, error) {
	entry, err := s.redis.GetPlayerRank(ctx, leaderboardID, playerID)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// GetAroundPlayer returns players around a specific player's rank
func (s *LeaderboardService) GetAroundPlayer(ctx context.Context, leaderboardID, playerID string, count int) ([]domain.LeaderboardEntry, error) {
	if count <= 0 {
		count = 5
	}
	if count > 50 {
		count = 50
	}

	entries, err := s.redis.GetAroundPlayer(ctx, leaderboardID, playerID, count)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GetRange returns players within a specific rank range
func (s *LeaderboardService) GetRange(ctx context.Context, leaderboardID string, start, end int) ([]domain.LeaderboardEntry, error) {
	// Validate range
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end-start > s.config.MaxLimit {
		end = start + s.config.MaxLimit
	}

	entries, err := s.redis.GetRange(ctx, leaderboardID, start, end)
	if err != nil {
		return nil, fmt.Errorf("getting range from redis: %w", err)
	}
	return entries, nil
}

// GetCount returns the total number of players in a leaderboard
func (s *LeaderboardService) GetCount(ctx context.Context, leaderboardID string) (int64, error) {
	return s.redis.GetCount(ctx, leaderboardID)
}

// RemovePlayer removes a player from a leaderboard
func (s *LeaderboardService) RemovePlayer(ctx context.Context, leaderboardID, playerID string) error {
	// Remove from Redis
	if err := s.redis.RemovePlayer(ctx, leaderboardID, playerID); err != nil {
		return fmt.Errorf("removing from redis: %w", err)
	}

	// Remove from PostgreSQL
	if err := s.postgres.RemovePlayer(ctx, leaderboardID, playerID); err != nil {
		// Log but don't fail if PostgreSQL removal fails
		s.logger.Warn("failed to remove player from postgres", "error", err)
	}

	return nil
}

// CreateLeaderboard creates a new leaderboard
func (s *LeaderboardService) CreateLeaderboard(ctx context.Context, req domain.CreateLeaderboardRequest) (*domain.LeaderboardConfig, error) {
	// Validate request
	if req.ID == "" || req.Name == "" {
		return nil, domain.ErrInvalidLeaderboard
	}

	// Check if leaderboard exists
	exists, err := s.postgres.LeaderboardExists(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("checking leaderboard existence: %w", err)
	}
	if exists {
		return nil, domain.ErrLeaderboardExists
	}

	// Convert to config with defaults
	config := req.ToConfig()

	// Create in PostgreSQL
	if err := s.postgres.CreateLeaderboard(ctx, config); err != nil {
		return nil, fmt.Errorf("creating leaderboard in postgres: %w", err)
	}

	// Store metadata in Redis
	if err := s.redis.SetLeaderboardMeta(ctx, config); err != nil {
		s.logger.Warn("failed to store leaderboard meta in redis", "error", err)
	}

	return &config, nil
}

// ListLeaderboards returns all leaderboards
func (s *LeaderboardService) ListLeaderboards(ctx context.Context) ([]domain.LeaderboardConfig, error) {
	return s.postgres.ListLeaderboards(ctx)
}

// GetLeaderboard returns a leaderboard by ID
func (s *LeaderboardService) GetLeaderboard(ctx context.Context, leaderboardID string) (*domain.LeaderboardConfig, error) {
	return s.postgres.GetLeaderboard(ctx, leaderboardID)
}

// DeleteLeaderboard deletes a leaderboard
func (s *LeaderboardService) DeleteLeaderboard(ctx context.Context, leaderboardID string) error {
	// Delete from Redis
	if err := s.redis.DeleteLeaderboard(ctx, leaderboardID); err != nil {
		s.logger.Warn("failed to delete leaderboard from redis", "error", err)
	}

	// Delete from PostgreSQL
	if err := s.postgres.DeleteLeaderboard(ctx, leaderboardID); err != nil {
		return fmt.Errorf("deleting leaderboard from postgres: %w", err)
	}

	return nil
}

// ResetLeaderboard clears all scores from a leaderboard
func (s *LeaderboardService) ResetLeaderboard(ctx context.Context, leaderboardID string) error {
	// Check if leaderboard exists
	exists, err := s.postgres.LeaderboardExists(ctx, leaderboardID)
	if err != nil {
		return fmt.Errorf("checking leaderboard existence: %w", err)
	}
	if !exists {
		return domain.ErrLeaderboardNotFound
	}

	// Reset in Redis
	if err := s.redis.ResetLeaderboard(ctx, leaderboardID); err != nil {
		return fmt.Errorf("resetting leaderboard in redis: %w", err)
	}

	// Reset in PostgreSQL
	if err := s.postgres.ResetLeaderboard(ctx, leaderboardID); err != nil {
		return fmt.Errorf("resetting leaderboard in postgres: %w", err)
	}

	return nil
}

// GetStats returns statistics for a leaderboard
func (s *LeaderboardService) GetStats(ctx context.Context, leaderboardID string) (*domain.LeaderboardStats, error) {
	count, err := s.redis.GetCount(ctx, leaderboardID)
	if err != nil {
		return nil, fmt.Errorf("getting count: %w", err)
	}

	stats := &domain.LeaderboardStats{
		LeaderboardID: leaderboardID,
		TotalPlayers:  count,
	}

	// Get top score
	top, err := s.redis.GetTopN(ctx, leaderboardID, 1)
	if err == nil && len(top) > 0 {
		stats.TopScore = top[0].Score
	}

	// Get bottom score
	bottom, err := s.redis.GetBottomN(ctx, leaderboardID, 1)
	if err == nil && len(bottom) > 0 {
		stats.LowestScore = bottom[0].Score
	}

	return stats, nil
}

