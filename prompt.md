# Leaderboard System - Development Prompt

## Project Overview

Build a real-time leaderboard application in Go with two data paths:
1. **Real-time path**: Instant updates with eventual consistency (Redis)
2. **Batch path**: 30-minute delayed updates with full accuracy (PostgreSQL)

## Tech Stack

- **Language**: Go 1.21+
- **Real-time Store**: Redis (Sorted Sets)
- **Persistent Store**: PostgreSQL
- **Message Queue** (optional): Redis Streams or Kafka for high-volume scenarios

## Architecture

```
┌─────────────┐     ┌─────────────────────────────────────┐
│   Client    │────►│           Go API Server             │
└─────────────┘     └──────────────┬──────────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    ▼                             ▼
          ┌─────────────────┐           ┌─────────────────┐
          │     Redis       │           │   PostgreSQL    │
          │  (Real-time)    │           │    (Batch)      │
          │  Sorted Sets    │           │   Source of     │
          │                 │           │     Truth       │
          └─────────────────┘           └─────────────────┘
                    ▲                             │
                    │         ┌───────────┐       │
                    └─────────│  Worker   │◄──────┘
                              │ (30 min)  │
                              └───────────┘
```

## Core Data Structures

### Player Score Event
```go
type ScoreEvent struct {
    PlayerID    string    `json:"player_id"`
    Score       int64     `json:"score"`
    GameID      string    `json:"game_id"`
    Timestamp   time.Time `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

### Leaderboard Entry
```go
type LeaderboardEntry struct {
    Rank      int64   `json:"rank"`
    PlayerID  string  `json:"player_id"`
    Score     int64   `json:"score"`
    Username  string  `json:"username,omitempty"`
}
```

### Leaderboard Configuration
```go
type LeaderboardConfig struct {
    ID              string        `json:"id"`
    Name            string        `json:"name"`
    SortOrder       string        `json:"sort_order"` // "desc" or "asc"
    ResetPeriod     string        `json:"reset_period"` // "daily", "weekly", "monthly", "never"
    MaxEntries      int           `json:"max_entries"`
    UpdateMode      string        `json:"update_mode"` // "replace", "increment", "best"
}
```

## API Endpoints

### Write Operations
```
POST   /api/v1/scores                    # Submit a score
POST   /api/v1/scores/batch              # Submit multiple scores
DELETE /api/v1/leaderboards/{id}/player/{player_id}  # Remove player
```

### Read Operations
```
GET    /api/v1/leaderboards/{id}/top?limit=10           # Get top N
GET    /api/v1/leaderboards/{id}/around/{player_id}?range=5  # Get surrounding ranks
GET    /api/v1/leaderboards/{id}/player/{player_id}     # Get player rank & score
GET    /api/v1/leaderboards/{id}/range?start=10&end=20  # Get rank range
```

### Admin Operations
```
POST   /api/v1/leaderboards              # Create leaderboard
GET    /api/v1/leaderboards              # List leaderboards
DELETE /api/v1/leaderboards/{id}         # Delete leaderboard
POST   /api/v1/leaderboards/{id}/reset   # Reset leaderboard
```

## Redis Commands Reference

### Key Patterns
```
leaderboard:{id}:realtime     # Sorted set for real-time scores
leaderboard:{id}:meta         # Hash for leaderboard metadata
player:{player_id}:info       # Hash for player info cache
```

### Core Operations
```redis
# Add/Update score (higher is better)
ZADD leaderboard:game1:realtime 1500 "player:123"

# Add with NX (only if not exists) or XX (only if exists)
ZADD leaderboard:game1:realtime XX 1600 "player:123"

# Increment score
ZINCRBY leaderboard:game1:realtime 100 "player:123"

# Get top 10 (descending)
ZREVRANGE leaderboard:game1:realtime 0 9 WITHSCORES

# Get player rank (0-indexed, descending)
ZREVRANK leaderboard:game1:realtime "player:123"

# Get player score
ZSCORE leaderboard:game1:realtime "player:123"

# Get players around a rank
ZREVRANGE leaderboard:game1:realtime 5 15 WITHSCORES

# Count total players
ZCARD leaderboard:game1:realtime

# Remove player
ZREM leaderboard:game1:realtime "player:123"
```

## PostgreSQL Schema

```sql
-- Leaderboards table
CREATE TABLE leaderboards (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sort_order VARCHAR(10) DEFAULT 'desc',
    reset_period VARCHAR(20) DEFAULT 'never',
    max_entries INT DEFAULT 10000,
    update_mode VARCHAR(20) DEFAULT 'replace',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Player scores (source of truth)
CREATE TABLE player_scores (
    id BIGSERIAL PRIMARY KEY,
    leaderboard_id VARCHAR(64) NOT NULL REFERENCES leaderboards(id),
    player_id VARCHAR(64) NOT NULL,
    score BIGINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(leaderboard_id, player_id)
);

-- Score history for auditing
CREATE TABLE score_events (
    id BIGSERIAL PRIMARY KEY,
    leaderboard_id VARCHAR(64) NOT NULL,
    player_id VARCHAR(64) NOT NULL,
    score BIGINT NOT NULL,
    event_type VARCHAR(20) NOT NULL, -- 'submit', 'increment', 'reset'
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_player_scores_leaderboard ON player_scores(leaderboard_id);
CREATE INDEX idx_player_scores_score ON player_scores(leaderboard_id, score DESC);
CREATE INDEX idx_score_events_player ON score_events(player_id, created_at DESC);
```

## Implementation Requirements

### 1. Redis Client Service
```go
type RedisLeaderboard interface {
    // Write
    SetScore(ctx context.Context, leaderboardID, playerID string, score int64) error
    IncrementScore(ctx context.Context, leaderboardID, playerID string, delta int64) (int64, error)
    RemovePlayer(ctx context.Context, leaderboardID, playerID string) error
    
    // Read
    GetTopN(ctx context.Context, leaderboardID string, n int) ([]LeaderboardEntry, error)
    GetPlayerRank(ctx context.Context, leaderboardID, playerID string) (*LeaderboardEntry, error)
    GetAroundPlayer(ctx context.Context, leaderboardID, playerID string, count int) ([]LeaderboardEntry, error)
    GetRange(ctx context.Context, leaderboardID string, start, end int) ([]LeaderboardEntry, error)
    GetCount(ctx context.Context, leaderboardID string) (int64, error)
}
```

### 2. PostgreSQL Repository
```go
type ScoreRepository interface {
    // Write
    UpsertScore(ctx context.Context, leaderboardID, playerID string, score int64, metadata map[string]interface{}) error
    RecordEvent(ctx context.Context, event ScoreEvent) error
    
    // Read
    GetLeaderboard(ctx context.Context, leaderboardID string, limit, offset int) ([]LeaderboardEntry, error)
    GetPlayerScore(ctx context.Context, leaderboardID, playerID string) (*LeaderboardEntry, error)
    
    // Admin
    CreateLeaderboard(ctx context.Context, config LeaderboardConfig) error
    ResetLeaderboard(ctx context.Context, leaderboardID string) error
}
```

### 3. Sync Worker
```go
type SyncWorker interface {
    // Sync Redis to PostgreSQL (runs every 30 minutes)
    SyncToDatabase(ctx context.Context, leaderboardID string) error
    
    // Sync PostgreSQL to Redis (for recovery/initialization)
    SyncFromDatabase(ctx context.Context, leaderboardID string) error
    
    // Start background sync
    Start(ctx context.Context) error
    Stop() error
}
```

### 4. HTTP Handlers
Use standard library `net/http` or a lightweight router like `chi` or `gorilla/mux`.

## Configuration

```yaml
# config.yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 100

postgres:
  host: "localhost"
  port: 5432
  user: "leaderboard"
  password: "secret"
  database: "leaderboard"
  max_connections: 50

sync:
  interval: 30m
  batch_size: 1000

leaderboard:
  default_limit: 100
  max_limit: 1000
```

## Error Handling

Define custom errors:
```go
var (
    ErrPlayerNotFound      = errors.New("player not found in leaderboard")
    ErrLeaderboardNotFound = errors.New("leaderboard not found")
    ErrInvalidScore        = errors.New("invalid score value")
    ErrRateLimited         = errors.New("rate limit exceeded")
)
```

## Testing Requirements

1. Unit tests for all service methods
2. Integration tests with Redis and PostgreSQL (use testcontainers-go)
3. Benchmark tests for high-volume scenarios
4. Example test cases:
   - Add 10,000 players, verify ranking
   - Concurrent score updates
   - Sync worker accuracy

## Performance Considerations

1. Use connection pooling for both Redis and PostgreSQL
2. Implement request batching for bulk score updates
3. Use pipelining for multiple Redis commands
4. Consider sharding for very large leaderboards (>1M players)
5. Cache player metadata separately from scores

## Directory Structure

```
leaderboard/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   │   ├── leaderboard.go
│   │   └── player.go
│   ├── redis/
│   │   └── leaderboard.go
│   ├── postgres/
│   │   ├── repository.go
│   │   └── migrations/
│   ├── service/
│   │   └── leaderboard.go
│   ├── handler/
│   │   └── http.go
│   └── worker/
│       └── sync.go
├── pkg/
│   └── client/
│       └── client.go
├── config.yaml
├── docker-compose.yaml
├── go.mod
└── README.md
```

## Sample Usage Flow

```go
// 1. Player submits score
client.SubmitScore("game1", "player123", 1500)

// 2. Real-time leaderboard updates instantly in Redis
// 3. Every 30 minutes, worker syncs to PostgreSQL

// 4. Get real-time top 10
top10 := client.GetTopPlayers("game1", 10)

// 5. Get player's current rank
rank := client.GetPlayerRank("game1", "player123")
```

## Docker Compose for Local Development

```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
      
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: leaderboard
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: leaderboard
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  redis_data:
  postgres_data:
```

## Deliverables

1. Complete Go application with all components
2. Docker Compose for local development
3. Sample client library in `pkg/client/`
4. Unit and integration tests
5. README with setup instructions
6. API documentation

## Notes for Implementation

- Use `github.com/redis/go-redis/v9` for Redis client
- Use `github.com/jackc/pgx/v5` for PostgreSQL
- Use `github.com/go-chi/chi/v5` for HTTP routing (optional)
- Implement graceful shutdown
- Add structured logging with `log/slog`
- Include health check endpoints