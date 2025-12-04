package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/domain"
	"github.com/redis/go-redis/v9"
)

// LeaderboardService provides Redis-based leaderboard operations
type LeaderboardService struct {
	client *redis.Client
	logger *slog.Logger
}

// NewLeaderboardService creates a new Redis leaderboard service
func NewLeaderboardService(cfg *config.RedisConfig, logger *slog.Logger) (*LeaderboardService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return &LeaderboardService{
		client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (s *LeaderboardService) Close() error {
	return s.client.Close()
}

// Client returns the underlying Redis client
func (s *LeaderboardService) Client() *redis.Client {
	return s.client
}

// leaderboardKey returns the Redis key for a leaderboard's sorted set
func (s *LeaderboardService) leaderboardKey(leaderboardID string) string {
	return fmt.Sprintf("leaderboard:%s:realtime", leaderboardID)
}

// metaKey returns the Redis key for leaderboard metadata
func (s *LeaderboardService) metaKey(leaderboardID string) string {
	return fmt.Sprintf("leaderboard:%s:meta", leaderboardID)
}

// playerInfoKey returns the Redis key for player info cache
func (s *LeaderboardService) playerInfoKey(playerID string) string {
	return fmt.Sprintf("player:%s:info", playerID)
}

// SetScore sets a player's score in the leaderboard
func (s *LeaderboardService) SetScore(ctx context.Context, leaderboardID, playerID string, score int64) error {
	key := s.leaderboardKey(leaderboardID)
	err := s.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(score),
		Member: playerID,
	}).Err()
	if err != nil {
		return fmt.Errorf("setting score: %w", err)
	}
	return nil
}

// SetScoreIfBetter sets a player's score only if it's better than the current score
func (s *LeaderboardService) SetScoreIfBetter(ctx context.Context, leaderboardID, playerID string, score int64, higherIsBetter bool) (bool, error) {
	key := s.leaderboardKey(leaderboardID)

	// Get current score
	currentScore, err := s.client.ZScore(ctx, key, playerID).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("getting current score: %w", err)
	}

	// If player doesn't exist, add them
	if err == redis.Nil {
		return true, s.SetScore(ctx, leaderboardID, playerID, score)
	}

	// Check if new score is better
	isBetter := (higherIsBetter && float64(score) > currentScore) ||
		(!higherIsBetter && float64(score) < currentScore)

	if !isBetter {
		return false, nil
	}

	return true, s.SetScore(ctx, leaderboardID, playerID, score)
}

// IncrementScore increments a player's score by the given delta
func (s *LeaderboardService) IncrementScore(ctx context.Context, leaderboardID, playerID string, delta int64) (int64, error) {
	key := s.leaderboardKey(leaderboardID)
	newScore, err := s.client.ZIncrBy(ctx, key, float64(delta), playerID).Result()
	if err != nil {
		return 0, fmt.Errorf("incrementing score: %w", err)
	}
	return int64(newScore), nil
}

// RemovePlayer removes a player from the leaderboard
func (s *LeaderboardService) RemovePlayer(ctx context.Context, leaderboardID, playerID string) error {
	key := s.leaderboardKey(leaderboardID)
	err := s.client.ZRem(ctx, key, playerID).Err()
	if err != nil {
		return fmt.Errorf("removing player: %w", err)
	}
	return nil
}

// GetTopN returns the top N players from the leaderboard (descending order)
func (s *LeaderboardService) GetTopN(ctx context.Context, leaderboardID string, n int) ([]domain.LeaderboardEntry, error) {
	key := s.leaderboardKey(leaderboardID)
	results, err := s.client.ZRevRangeWithScores(ctx, key, 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("getting top n: %w", err)
	}

	entries := make([]domain.LeaderboardEntry, len(results))
	for i, result := range results {
		entries[i] = domain.LeaderboardEntry{
			Rank:     int64(i + 1),
			PlayerID: result.Member.(string),
			Score:    int64(result.Score),
		}
	}
	return entries, nil
}

// GetBottomN returns the bottom N players from the leaderboard (ascending order)
func (s *LeaderboardService) GetBottomN(ctx context.Context, leaderboardID string, n int) ([]domain.LeaderboardEntry, error) {
	key := s.leaderboardKey(leaderboardID)
	totalCount, err := s.client.ZCard(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("getting count: %w", err)
	}

	results, err := s.client.ZRangeWithScores(ctx, key, 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("getting bottom n: %w", err)
	}

	entries := make([]domain.LeaderboardEntry, len(results))
	for i, result := range results {
		entries[i] = domain.LeaderboardEntry{
			Rank:     totalCount - int64(i),
			PlayerID: result.Member.(string),
			Score:    int64(result.Score),
		}
	}
	return entries, nil
}

// GetPlayerRank returns a player's rank and score
func (s *LeaderboardService) GetPlayerRank(ctx context.Context, leaderboardID, playerID string) (*domain.LeaderboardEntry, error) {
	key := s.leaderboardKey(leaderboardID)

	// Use pipeline to get both rank and score
	pipe := s.client.Pipeline()
	rankCmd := pipe.ZRevRank(ctx, key, playerID)
	scoreCmd := pipe.ZScore(ctx, key, playerID)
	_, err := pipe.Exec(ctx)

	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrPlayerNotFound
		}
		return nil, fmt.Errorf("getting player rank: %w", err)
	}

	rank, err := rankCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrPlayerNotFound
		}
		return nil, fmt.Errorf("getting rank result: %w", err)
	}

	score, err := scoreCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("getting score result: %w", err)
	}

	return &domain.LeaderboardEntry{
		Rank:     rank + 1, // Convert 0-indexed to 1-indexed
		PlayerID: playerID,
		Score:    int64(score),
	}, nil
}

// GetAroundPlayer returns players around a specific player's rank
func (s *LeaderboardService) GetAroundPlayer(ctx context.Context, leaderboardID, playerID string, count int) ([]domain.LeaderboardEntry, error) {
	// First, get the player's rank
	playerEntry, err := s.GetPlayerRank(ctx, leaderboardID, playerID)
	if err != nil {
		return nil, err
	}

	// Calculate range around the player
	start := playerEntry.Rank - int64(count) - 1 // -1 because rank is 1-indexed
	if start < 0 {
		start = 0
	}
	end := playerEntry.Rank + int64(count) - 1 // -1 because rank is 1-indexed

	return s.GetRange(ctx, leaderboardID, int(start), int(end))
}

// GetRange returns players within a specific rank range (0-indexed)
func (s *LeaderboardService) GetRange(ctx context.Context, leaderboardID string, start, end int) ([]domain.LeaderboardEntry, error) {
	key := s.leaderboardKey(leaderboardID)
	results, err := s.client.ZRevRangeWithScores(ctx, key, int64(start), int64(end)).Result()
	if err != nil {
		return nil, fmt.Errorf("getting range: %w", err)
	}

	entries := make([]domain.LeaderboardEntry, len(results))
	for i, result := range results {
		entries[i] = domain.LeaderboardEntry{
			Rank:     int64(start + i + 1), // Convert to 1-indexed rank
			PlayerID: result.Member.(string),
			Score:    int64(result.Score),
		}
	}
	return entries, nil
}

// GetCount returns the total number of players in the leaderboard
func (s *LeaderboardService) GetCount(ctx context.Context, leaderboardID string) (int64, error) {
	key := s.leaderboardKey(leaderboardID)
	count, err := s.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("getting count: %w", err)
	}
	return count, nil
}

// GetAllScores returns all players and scores from the leaderboard
func (s *LeaderboardService) GetAllScores(ctx context.Context, leaderboardID string) ([]domain.LeaderboardEntry, error) {
	key := s.leaderboardKey(leaderboardID)
	results, err := s.client.ZRevRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("getting all scores: %w", err)
	}

	entries := make([]domain.LeaderboardEntry, len(results))
	for i, result := range results {
		entries[i] = domain.LeaderboardEntry{
			Rank:     int64(i + 1),
			PlayerID: result.Member.(string),
			Score:    int64(result.Score),
		}
	}
	return entries, nil
}

// DeleteLeaderboard removes an entire leaderboard
func (s *LeaderboardService) DeleteLeaderboard(ctx context.Context, leaderboardID string) error {
	key := s.leaderboardKey(leaderboardID)
	metaKey := s.metaKey(leaderboardID)

	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, metaKey)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("deleting leaderboard: %w", err)
	}
	return nil
}

// ResetLeaderboard clears all entries from a leaderboard
func (s *LeaderboardService) ResetLeaderboard(ctx context.Context, leaderboardID string) error {
	key := s.leaderboardKey(leaderboardID)
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("resetting leaderboard: %w", err)
	}
	return nil
}

// SetLeaderboardMeta stores leaderboard metadata
func (s *LeaderboardService) SetLeaderboardMeta(ctx context.Context, config domain.LeaderboardConfig) error {
	key := s.metaKey(config.ID)
	err := s.client.HSet(ctx, key,
		"id", config.ID,
		"name", config.Name,
		"sort_order", string(config.SortOrder),
		"reset_period", string(config.ResetPeriod),
		"max_entries", config.MaxEntries,
		"update_mode", string(config.UpdateMode),
	).Err()
	if err != nil {
		return fmt.Errorf("setting leaderboard meta: %w", err)
	}
	return nil
}

// GetLeaderboardMeta retrieves leaderboard metadata
func (s *LeaderboardService) GetLeaderboardMeta(ctx context.Context, leaderboardID string) (*domain.LeaderboardConfig, error) {
	key := s.metaKey(leaderboardID)
	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("getting leaderboard meta: %w", err)
	}

	if len(result) == 0 {
		return nil, domain.ErrLeaderboardNotFound
	}

	maxEntries, _ := strconv.Atoi(result["max_entries"])

	return &domain.LeaderboardConfig{
		ID:          result["id"],
		Name:        result["name"],
		SortOrder:   domain.SortOrder(result["sort_order"]),
		ResetPeriod: domain.ResetPeriod(result["reset_period"]),
		MaxEntries:  maxEntries,
		UpdateMode:  domain.UpdateMode(result["update_mode"]),
	}, nil
}

// SetPlayerInfo caches player information
func (s *LeaderboardService) SetPlayerInfo(ctx context.Context, playerID, username string) error {
	key := s.playerInfoKey(playerID)
	err := s.client.HSet(ctx, key, "username", username).Err()
	if err != nil {
		return fmt.Errorf("setting player info: %w", err)
	}
	return nil
}

// GetPlayerInfo retrieves cached player information
func (s *LeaderboardService) GetPlayerInfo(ctx context.Context, playerID string) (*domain.PlayerInfo, error) {
	key := s.playerInfoKey(playerID)
	result, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("getting player info: %w", err)
	}

	if len(result) == 0 {
		return nil, domain.ErrPlayerNotFound
	}

	return &domain.PlayerInfo{
		ID:       playerID,
		Username: result["username"],
	}, nil
}

// BatchSetScores sets multiple scores using pipelining
func (s *LeaderboardService) BatchSetScores(ctx context.Context, leaderboardID string, scores map[string]int64) error {
	key := s.leaderboardKey(leaderboardID)
	pipe := s.client.Pipeline()

	for playerID, score := range scores {
		pipe.ZAdd(ctx, key, redis.Z{
			Score:  float64(score),
			Member: playerID,
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("batch setting scores: %w", err)
	}
	return nil
}

// Exists checks if a leaderboard exists in Redis
func (s *LeaderboardService) Exists(ctx context.Context, leaderboardID string) (bool, error) {
	key := s.leaderboardKey(leaderboardID)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("checking existence: %w", err)
	}
	return exists > 0, nil
}

