package domain

import "time"

// Player represents a player in the system
type Player struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlayerInfo is a lightweight player information struct used for caching
type PlayerInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// PlayerScore represents a player's score in a specific leaderboard
type PlayerScore struct {
	PlayerID      string                 `json:"player_id"`
	LeaderboardID string                 `json:"leaderboard_id"`
	Score         int64                  `json:"score"`
	Rank          int64                  `json:"rank"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

