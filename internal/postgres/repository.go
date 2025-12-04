package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/domain"
)

// Repository provides PostgreSQL-based data access
type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewRepository creates a new PostgreSQL repository
func NewRepository(cfg *config.PostgresConfig, logger *slog.Logger) (*Repository, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return &Repository{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (r *Repository) Close() {
	r.pool.Close()
}

// Pool returns the underlying connection pool
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// RunMigrations executes database migrations
func (r *Repository) RunMigrations(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS leaderboards (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			sort_order VARCHAR(10) DEFAULT 'desc',
			reset_period VARCHAR(20) DEFAULT 'never',
			max_entries INT DEFAULT 10000,
			update_mode VARCHAR(20) DEFAULT 'replace',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS player_scores (
			id BIGSERIAL PRIMARY KEY,
			leaderboard_id VARCHAR(64) NOT NULL REFERENCES leaderboards(id) ON DELETE CASCADE,
			player_id VARCHAR(64) NOT NULL,
			score BIGINT NOT NULL,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(leaderboard_id, player_id)
		)`,
		`CREATE TABLE IF NOT EXISTS score_events (
			id BIGSERIAL PRIMARY KEY,
			leaderboard_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			score BIGINT NOT NULL,
			event_type VARCHAR(20) NOT NULL,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_player_scores_leaderboard ON player_scores(leaderboard_id)`,
		`CREATE INDEX IF NOT EXISTS idx_player_scores_score ON player_scores(leaderboard_id, score DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_score_events_player ON score_events(player_id, created_at DESC)`,
	}

	for _, migration := range migrations {
		_, err := r.pool.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("executing migration: %w", err)
		}
	}

	r.logger.Info("database migrations completed")
	return nil
}

// CreateLeaderboard creates a new leaderboard configuration
func (r *Repository) CreateLeaderboard(ctx context.Context, config domain.LeaderboardConfig) error {
	query := `
		INSERT INTO leaderboards (id, name, sort_order, reset_period, max_entries, update_mode, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	now := time.Now()
	_, err := r.pool.Exec(ctx, query,
		config.ID,
		config.Name,
		string(config.SortOrder),
		string(config.ResetPeriod),
		config.MaxEntries,
		string(config.UpdateMode),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("creating leaderboard: %w", err)
	}
	return nil
}

// GetLeaderboard retrieves a leaderboard configuration by ID
func (r *Repository) GetLeaderboard(ctx context.Context, leaderboardID string) (*domain.LeaderboardConfig, error) {
	query := `
		SELECT id, name, sort_order, reset_period, max_entries, update_mode, created_at, updated_at
		FROM leaderboards
		WHERE id = $1
	`
	var config domain.LeaderboardConfig
	err := r.pool.QueryRow(ctx, query, leaderboardID).Scan(
		&config.ID,
		&config.Name,
		&config.SortOrder,
		&config.ResetPeriod,
		&config.MaxEntries,
		&config.UpdateMode,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrLeaderboardNotFound
		}
		return nil, fmt.Errorf("getting leaderboard: %w", err)
	}
	return &config, nil
}

// ListLeaderboards retrieves all leaderboard configurations
func (r *Repository) ListLeaderboards(ctx context.Context) ([]domain.LeaderboardConfig, error) {
	query := `
		SELECT id, name, sort_order, reset_period, max_entries, update_mode, created_at, updated_at
		FROM leaderboards
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing leaderboards: %w", err)
	}
	defer rows.Close()

	var configs []domain.LeaderboardConfig
	for rows.Next() {
		var config domain.LeaderboardConfig
		err := rows.Scan(
			&config.ID,
			&config.Name,
			&config.SortOrder,
			&config.ResetPeriod,
			&config.MaxEntries,
			&config.UpdateMode,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning leaderboard: %w", err)
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// DeleteLeaderboard removes a leaderboard and all associated data
func (r *Repository) DeleteLeaderboard(ctx context.Context, leaderboardID string) error {
	query := `DELETE FROM leaderboards WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, leaderboardID)
	if err != nil {
		return fmt.Errorf("deleting leaderboard: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrLeaderboardNotFound
	}
	return nil
}

// UpsertScore inserts or updates a player's score
func (r *Repository) UpsertScore(ctx context.Context, leaderboardID, playerID string, score int64, metadata map[string]interface{}) error {
	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshaling metadata: %w", err)
		}
	}

	query := `
		INSERT INTO player_scores (leaderboard_id, player_id, score, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (leaderboard_id, player_id) 
		DO UPDATE SET score = $3, metadata = COALESCE($4, player_scores.metadata), updated_at = $5
	`
	now := time.Now()
	_, err = r.pool.Exec(ctx, query, leaderboardID, playerID, score, metadataJSON, now)
	if err != nil {
		return fmt.Errorf("upserting score: %w", err)
	}
	return nil
}

// UpsertScoreBest updates score only if it's better than the existing one
func (r *Repository) UpsertScoreBest(ctx context.Context, leaderboardID, playerID string, score int64, higherIsBetter bool, metadata map[string]interface{}) error {
	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshaling metadata: %w", err)
		}
	}

	var query string
	if higherIsBetter {
		query = `
			INSERT INTO player_scores (leaderboard_id, player_id, score, metadata, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
			ON CONFLICT (leaderboard_id, player_id) 
			DO UPDATE SET 
				score = GREATEST(player_scores.score, $3),
				metadata = COALESCE($4, player_scores.metadata),
				updated_at = $5
		`
	} else {
		query = `
			INSERT INTO player_scores (leaderboard_id, player_id, score, metadata, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5)
			ON CONFLICT (leaderboard_id, player_id) 
			DO UPDATE SET 
				score = LEAST(player_scores.score, $3),
				metadata = COALESCE($4, player_scores.metadata),
				updated_at = $5
		`
	}
	now := time.Now()
	_, err = r.pool.Exec(ctx, query, leaderboardID, playerID, score, metadataJSON, now)
	if err != nil {
		return fmt.Errorf("upserting best score: %w", err)
	}
	return nil
}

// IncrementScore increments a player's score by the given delta
func (r *Repository) IncrementScore(ctx context.Context, leaderboardID, playerID string, delta int64) (int64, error) {
	query := `
		INSERT INTO player_scores (leaderboard_id, player_id, score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (leaderboard_id, player_id) 
		DO UPDATE SET score = player_scores.score + $3, updated_at = $4
		RETURNING score
	`
	now := time.Now()
	var newScore int64
	err := r.pool.QueryRow(ctx, query, leaderboardID, playerID, delta, now).Scan(&newScore)
	if err != nil {
		return 0, fmt.Errorf("incrementing score: %w", err)
	}
	return newScore, nil
}

// RecordEvent records a score event for auditing
func (r *Repository) RecordEvent(ctx context.Context, event domain.ScoreEvent) error {
	var metadataJSON []byte
	var err error
	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshaling metadata: %w", err)
		}
	}

	query := `
		INSERT INTO score_events (leaderboard_id, player_id, score, event_type, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = r.pool.Exec(ctx, query,
		event.LeaderboardID,
		event.PlayerID,
		event.Score,
		event.EventType,
		metadataJSON,
		event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("recording event: %w", err)
	}
	return nil
}

// GetLeaderboardEntries retrieves leaderboard entries with pagination
func (r *Repository) GetLeaderboardEntries(ctx context.Context, leaderboardID string, limit, offset int, descending bool) ([]domain.LeaderboardEntry, error) {
	var query string
	if descending {
		query = `
			SELECT player_id, score, 
				   ROW_NUMBER() OVER (ORDER BY score DESC) as rank
			FROM player_scores
			WHERE leaderboard_id = $1
			ORDER BY score DESC
			LIMIT $2 OFFSET $3
		`
	} else {
		query = `
			SELECT player_id, score, 
				   ROW_NUMBER() OVER (ORDER BY score ASC) as rank
			FROM player_scores
			WHERE leaderboard_id = $1
			ORDER BY score ASC
			LIMIT $2 OFFSET $3
		`
	}

	rows, err := r.pool.Query(ctx, query, leaderboardID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("getting leaderboard entries: %w", err)
	}
	defer rows.Close()

	var entries []domain.LeaderboardEntry
	for rows.Next() {
		var entry domain.LeaderboardEntry
		err := rows.Scan(&entry.PlayerID, &entry.Score, &entry.Rank)
		if err != nil {
			return nil, fmt.Errorf("scanning entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// GetPlayerScore retrieves a player's score and rank
func (r *Repository) GetPlayerScore(ctx context.Context, leaderboardID, playerID string) (*domain.LeaderboardEntry, error) {
	query := `
		WITH ranked AS (
			SELECT player_id, score,
				   ROW_NUMBER() OVER (ORDER BY score DESC) as rank
			FROM player_scores
			WHERE leaderboard_id = $1
		)
		SELECT player_id, score, rank
		FROM ranked
		WHERE player_id = $2
	`
	var entry domain.LeaderboardEntry
	err := r.pool.QueryRow(ctx, query, leaderboardID, playerID).Scan(
		&entry.PlayerID,
		&entry.Score,
		&entry.Rank,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPlayerNotFound
		}
		return nil, fmt.Errorf("getting player score: %w", err)
	}
	return &entry, nil
}

// RemovePlayer removes a player from a leaderboard
func (r *Repository) RemovePlayer(ctx context.Context, leaderboardID, playerID string) error {
	query := `DELETE FROM player_scores WHERE leaderboard_id = $1 AND player_id = $2`
	result, err := r.pool.Exec(ctx, query, leaderboardID, playerID)
	if err != nil {
		return fmt.Errorf("removing player: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrPlayerNotFound
	}
	return nil
}

// ResetLeaderboard clears all player scores for a leaderboard
func (r *Repository) ResetLeaderboard(ctx context.Context, leaderboardID string) error {
	query := `DELETE FROM player_scores WHERE leaderboard_id = $1`
	_, err := r.pool.Exec(ctx, query, leaderboardID)
	if err != nil {
		return fmt.Errorf("resetting leaderboard: %w", err)
	}
	return nil
}

// GetAllScores retrieves all player scores for a leaderboard (for sync)
func (r *Repository) GetAllScores(ctx context.Context, leaderboardID string) (map[string]int64, error) {
	query := `SELECT player_id, score FROM player_scores WHERE leaderboard_id = $1`
	rows, err := r.pool.Query(ctx, query, leaderboardID)
	if err != nil {
		return nil, fmt.Errorf("getting all scores: %w", err)
	}
	defer rows.Close()

	scores := make(map[string]int64)
	for rows.Next() {
		var playerID string
		var score int64
		if err := rows.Scan(&playerID, &score); err != nil {
			return nil, fmt.Errorf("scanning score: %w", err)
		}
		scores[playerID] = score
	}
	return scores, nil
}

// GetPlayerCount returns the total number of players in a leaderboard
func (r *Repository) GetPlayerCount(ctx context.Context, leaderboardID string) (int64, error) {
	query := `SELECT COUNT(*) FROM player_scores WHERE leaderboard_id = $1`
	var count int64
	err := r.pool.QueryRow(ctx, query, leaderboardID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting player count: %w", err)
	}
	return count, nil
}

// LeaderboardExists checks if a leaderboard exists
func (r *Repository) LeaderboardExists(ctx context.Context, leaderboardID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM leaderboards WHERE id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, leaderboardID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking leaderboard existence: %w", err)
	}
	return exists, nil
}

// BatchUpsertScores inserts or updates multiple scores efficiently
func (r *Repository) BatchUpsertScores(ctx context.Context, leaderboardID string, scores map[string]int64) error {
	if len(scores) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO player_scores (leaderboard_id, player_id, score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (leaderboard_id, player_id) 
		DO UPDATE SET score = $3, updated_at = $4
	`
	now := time.Now()

	for playerID, score := range scores {
		batch.Queue(query, leaderboardID, playerID, score, now)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range scores {
		_, err := br.Exec()
		if err != nil {
			return fmt.Errorf("batch upserting scores: %w", err)
		}
	}
	return nil
}

