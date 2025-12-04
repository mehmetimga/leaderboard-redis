package domain

import (
	"time"
)

// SortOrder represents the sort direction for leaderboard rankings
type SortOrder string

const (
	SortOrderDesc SortOrder = "desc"
	SortOrderAsc  SortOrder = "asc"
)

// ResetPeriod represents how often a leaderboard resets
type ResetPeriod string

const (
	ResetPeriodDaily   ResetPeriod = "daily"
	ResetPeriodWeekly  ResetPeriod = "weekly"
	ResetPeriodMonthly ResetPeriod = "monthly"
	ResetPeriodNever   ResetPeriod = "never"
)

// UpdateMode represents how scores are updated
type UpdateMode string

const (
	UpdateModeReplace   UpdateMode = "replace"
	UpdateModeIncrement UpdateMode = "increment"
	UpdateModeBest      UpdateMode = "best"
)

// LeaderboardConfig represents the configuration for a leaderboard
type LeaderboardConfig struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	SortOrder   SortOrder   `json:"sort_order"`
	ResetPeriod ResetPeriod `json:"reset_period"`
	MaxEntries  int         `json:"max_entries"`
	UpdateMode  UpdateMode  `json:"update_mode"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	Rank     int64  `json:"rank"`
	PlayerID string `json:"player_id"`
	Score    int64  `json:"score"`
	Username string `json:"username,omitempty"`
}

// ScoreEvent represents a score submission event
type ScoreEvent struct {
	PlayerID      string                 `json:"player_id"`
	LeaderboardID string                 `json:"leaderboard_id"`
	Score         int64                  `json:"score"`
	GameID        string                 `json:"game_id,omitempty"`
	EventType     string                 `json:"event_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ScoreSubmission represents a request to submit a score
type ScoreSubmission struct {
	PlayerID      string                 `json:"player_id"`
	LeaderboardID string                 `json:"leaderboard_id"`
	Score         int64                  `json:"score"`
	GameID        string                 `json:"game_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// BatchScoreSubmission represents multiple score submissions
type BatchScoreSubmission struct {
	Scores []ScoreSubmission `json:"scores"`
}

// CreateLeaderboardRequest represents a request to create a new leaderboard
type CreateLeaderboardRequest struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	SortOrder   SortOrder   `json:"sort_order,omitempty"`
	ResetPeriod ResetPeriod `json:"reset_period,omitempty"`
	MaxEntries  int         `json:"max_entries,omitempty"`
	UpdateMode  UpdateMode  `json:"update_mode,omitempty"`
}

// ToConfig converts a CreateLeaderboardRequest to a LeaderboardConfig with defaults
func (r *CreateLeaderboardRequest) ToConfig() LeaderboardConfig {
	config := LeaderboardConfig{
		ID:          r.ID,
		Name:        r.Name,
		SortOrder:   r.SortOrder,
		ResetPeriod: r.ResetPeriod,
		MaxEntries:  r.MaxEntries,
		UpdateMode:  r.UpdateMode,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Apply defaults
	if config.SortOrder == "" {
		config.SortOrder = SortOrderDesc
	}
	if config.ResetPeriod == "" {
		config.ResetPeriod = ResetPeriodNever
	}
	if config.MaxEntries == 0 {
		config.MaxEntries = 10000
	}
	if config.UpdateMode == "" {
		config.UpdateMode = UpdateModeReplace
	}

	return config
}

// LeaderboardStats contains statistics about a leaderboard
type LeaderboardStats struct {
	LeaderboardID string `json:"leaderboard_id"`
	TotalPlayers  int64  `json:"total_players"`
	TopScore      int64  `json:"top_score,omitempty"`
	LowestScore   int64  `json:"lowest_score,omitempty"`
}

