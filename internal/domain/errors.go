package domain

import "errors"

// Domain errors
var (
	ErrPlayerNotFound      = errors.New("player not found in leaderboard")
	ErrLeaderboardNotFound = errors.New("leaderboard not found")
	ErrLeaderboardExists   = errors.New("leaderboard already exists")
	ErrInvalidScore        = errors.New("invalid score value")
	ErrInvalidLeaderboard  = errors.New("invalid leaderboard configuration")
	ErrRateLimited         = errors.New("rate limit exceeded")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrInternalError       = errors.New("internal server error")
)

// IsNotFoundError checks if an error is a not-found type error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrPlayerNotFound) || errors.Is(err, ErrLeaderboardNotFound)
}

