# Leaderboard System

A real-time leaderboard application built in Go with dual data paths:
- **Real-time path**: Instant updates with eventual consistency (Redis)
- **Batch path**: 30-minute delayed updates with full accuracy (PostgreSQL)

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

## Tech Stack

- **Language**: Go 1.21+
- **Real-time Store**: Redis 7+ (Sorted Sets)
- **Persistent Store**: PostgreSQL 15+
- **HTTP Router**: Chi v5
- **Database Driver**: pgx v5

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services (Redis, PostgreSQL, and the server)
docker-compose up -d

# View logs
docker-compose logs -f server

# Stop services
docker-compose down
```

### Local Development

1. **Start Redis and PostgreSQL**:
```bash
# Start only Redis and PostgreSQL
docker-compose up -d redis postgres
```

2. **Build and run the server**:
```bash
# Download dependencies
go mod download

# Run the server
go run ./cmd/server -config config.yaml
```

3. **Or build a binary**:
```bash
go build -o leaderboard-server ./cmd/server
./leaderboard-server -config config.yaml
```

## API Endpoints

### Health Checks
- `GET /health` - Service health status
- `GET /ready` - Service readiness status

### Score Operations
- `POST /api/v1/scores` - Submit a score
- `POST /api/v1/scores/batch` - Submit multiple scores

### Leaderboard Management
- `POST /api/v1/leaderboards` - Create a leaderboard
- `GET /api/v1/leaderboards` - List all leaderboards
- `GET /api/v1/leaderboards/{id}` - Get leaderboard details
- `DELETE /api/v1/leaderboards/{id}` - Delete a leaderboard
- `POST /api/v1/leaderboards/{id}/reset` - Reset a leaderboard
- `GET /api/v1/leaderboards/{id}/stats` - Get leaderboard statistics

### Ranking Operations
- `GET /api/v1/leaderboards/{id}/top?limit=10` - Get top N players
- `GET /api/v1/leaderboards/{id}/range?start=10&end=20` - Get rank range
- `GET /api/v1/leaderboards/{id}/around/{player_id}?range=5` - Get surrounding ranks
- `GET /api/v1/leaderboards/{id}/player/{player_id}` - Get player rank & score
- `DELETE /api/v1/leaderboards/{id}/player/{player_id}` - Remove player

## API Usage Examples

### Create a Leaderboard

```bash
curl -X POST http://localhost:8080/api/v1/leaderboards \
  -H "Content-Type: application/json" \
  -d '{
    "id": "game1",
    "name": "Game 1 Leaderboard",
    "sort_order": "desc",
    "reset_period": "never",
    "max_entries": 10000,
    "update_mode": "best"
  }'
```

**Update Modes:**
- `replace` - Always replace the score
- `increment` - Add to existing score
- `best` - Keep the best score (highest for desc, lowest for asc)

### Submit a Score

```bash
curl -X POST http://localhost:8080/api/v1/scores \
  -H "Content-Type: application/json" \
  -d '{
    "player_id": "player123",
    "leaderboard_id": "game1",
    "score": 1500,
    "metadata": {"level": 10}
  }'
```

### Submit Batch Scores

```bash
curl -X POST http://localhost:8080/api/v1/scores/batch \
  -H "Content-Type: application/json" \
  -d '{
    "scores": [
      {"player_id": "player1", "leaderboard_id": "game1", "score": 1000},
      {"player_id": "player2", "leaderboard_id": "game1", "score": 2000},
      {"player_id": "player3", "leaderboard_id": "game1", "score": 1500}
    ]
  }'
```

### Get Top Players

```bash
curl http://localhost:8080/api/v1/leaderboards/game1/top?limit=10
```

**Response:**
```json
{
  "success": true,
  "data": [
    {"rank": 1, "player_id": "player2", "score": 2000},
    {"rank": 2, "player_id": "player3", "score": 1500},
    {"rank": 3, "player_id": "player1", "score": 1000}
  ]
}
```

### Get Player Rank

```bash
curl http://localhost:8080/api/v1/leaderboards/game1/player/player123
```

### Get Players Around a Player

```bash
curl http://localhost:8080/api/v1/leaderboards/game1/around/player123?range=5
```

### Get Rank Range

```bash
curl http://localhost:8080/api/v1/leaderboards/game1/range?start=10&end=20
```

## Configuration

Configuration is loaded from `config.yaml`. Environment variables can be used with the `${VAR:default}` syntax.

```yaml
server:
  port: 8080
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 120s

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
  interval: 30m      # Sync interval
  batch_size: 1000   # Batch size for sync operations
  enabled: true      # Enable/disable background sync

leaderboard:
  default_limit: 100
  max_limit: 1000
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | (empty) |
| `POSTGRES_HOST` | PostgreSQL host | `localhost` |
| `POSTGRES_USER` | PostgreSQL user | `leaderboard` |
| `POSTGRES_PASSWORD` | PostgreSQL password | `secret` |
| `POSTGRES_DB` | PostgreSQL database | `leaderboard` |

## Project Structure

```
leaderboard-redis/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration loading
│   ├── domain/
│   │   ├── leaderboard.go    # Domain types
│   │   ├── player.go         # Player types
│   │   └── errors.go         # Custom errors
│   ├── redis/
│   │   └── leaderboard.go    # Redis operations
│   ├── postgres/
│   │   └── repository.go     # PostgreSQL operations
│   ├── service/
│   │   └── leaderboard.go    # Business logic
│   ├── handler/
│   │   └── http.go           # HTTP handlers
│   └── worker/
│       └── sync.go           # Background sync worker
├── config.yaml               # Configuration file
├── docker-compose.yaml       # Docker Compose setup
├── Dockerfile                # Container build
├── go.mod                    # Go module
└── README.md                 # This file
```

## WebSocket Real-time Updates

The server supports WebSocket connections for real-time leaderboard updates.

### WebSocket Endpoint

```
ws://localhost:8080/ws
```

### Subscribe to Updates

```json
{"type": "subscribe", "leaderboard_id": "game1"}
```

### Receive Updates

```json
{
  "type": "leaderboard_update",
  "leaderboard_id": "game1",
  "data": {
    "entries": [{"rank": 1, "player_id": "player1", "score": 5000}, ...],
    "total_players": 1000
  }
}
```

## React Frontend

A React frontend is included in the `webapp/` directory:

```bash
cd webapp
npm install
npm run dev -- --port 3000
```

Features:
- Real-time leaderboard display
- Top 3 podium with medals
- Live connection indicator
- Rank change animations
- Auto-reconnect on disconnect

## Real-time Demo

For a complete demo with 1000 players and live updates, see:

📖 **[docs/REALTIME_DEMO.md](docs/REALTIME_DEMO.md)**

Quick start:
```bash
# Start everything
docker-compose up -d redis postgres
go run ./cmd/server -config config.yaml &
cd webapp && npm run dev -- --port 3000 &

# Run battle royale demo (1000 players, constant changes)
./scripts/battle-royale.sh

# Open http://localhost:3000 and watch!
```

## Performance Considerations

1. **Connection Pooling**: Both Redis and PostgreSQL use connection pools
2. **Pipelining**: Redis operations use pipelining for batch operations
3. **Batch Processing**: Sync worker processes scores in configurable batches
4. **Caching**: Player metadata is cached in Redis
5. **Indexes**: PostgreSQL uses indexes for efficient queries

## How It Works

### Real-time Updates (Redis)
When a score is submitted:
1. The score is immediately written to Redis sorted set
2. Redis provides O(log N) insertion and ranking queries
3. Players get instant feedback on their rank

### Batch Persistence (PostgreSQL)
Every 30 minutes (configurable):
1. Sync worker reads all scores from Redis
2. Scores are batch-upserted to PostgreSQL
3. PostgreSQL serves as the source of truth for historical data

### Recovery
On server startup:
1. All leaderboards are synced from PostgreSQL to Redis
2. This ensures Redis is populated after restarts
3. No data loss between Redis and PostgreSQL

## License

MIT

