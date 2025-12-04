# Leaderboard System - Testing Guide

## Prerequisites

- Docker and Docker Compose installed
- Go 1.21+ installed
- `curl` and `jq` for API testing

## Quick Start

### 1. Start Infrastructure

```bash
# Start Redis and PostgreSQL
docker-compose up -d redis postgres

# Verify services are healthy
docker-compose ps
```

Expected output:
```
NAME                   IMAGE                COMMAND                  SERVICE    STATUS
leaderboard-postgres   postgres:15-alpine   "docker-entrypoint.s…"   postgres   Up (healthy)
leaderboard-redis      redis:7-alpine       "docker-entrypoint.s…"   redis      Up (healthy)
```

### 2. Start the Server

```bash
# Run the server
go run ./cmd/server -config config.yaml
```

Expected output:
```json
{"time":"...","level":"INFO","msg":"connecting to Redis","addr":"localhost:6379"}
{"time":"...","level":"INFO","msg":"connected to Redis"}
{"time":"...","level":"INFO","msg":"connecting to PostgreSQL","host":"localhost","database":"leaderboard"}
{"time":"...","level":"INFO","msg":"connected to PostgreSQL"}
{"time":"...","level":"INFO","msg":"database migrations completed"}
{"time":"...","level":"INFO","msg":"syncing leaderboards from database to Redis"}
{"time":"...","level":"INFO","msg":"sync worker started","interval":1800000000000}
{"time":"...","level":"INFO","msg":"starting HTTP server","port":8080}
```

### 3. Run API Tests

Use the test script:
```bash
./scripts/test-api.sh
```

Or run individual tests manually (see below).

## Test Scripts

### Full API Test Script

Create and run `scripts/test-api.sh`:

```bash
#!/bin/bash
set -e

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Leaderboard API Test Suite"
echo "=========================================="

# Health Check
echo -e "\n--- Health Check ---"
curl -s "$BASE_URL/health" | jq .

# Create Leaderboard
echo -e "\n--- Create Leaderboard (game1 - best mode) ---"
curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "game1", "name": "Game 1 Leaderboard", "sort_order": "desc", "update_mode": "best"}' | jq .

# Submit Batch Scores
echo -e "\n--- Submit Batch Scores ---"
curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "scores": [
      {"player_id": "player1", "leaderboard_id": "game1", "score": 1000},
      {"player_id": "player2", "leaderboard_id": "game1", "score": 2500},
      {"player_id": "player3", "leaderboard_id": "game1", "score": 1800},
      {"player_id": "player4", "leaderboard_id": "game1", "score": 3200},
      {"player_id": "player5", "leaderboard_id": "game1", "score": 950}
    ]
  }' | jq .

# Get Top Players
echo -e "\n--- Get Top 10 Players ---"
curl -s "$BASE_URL/api/v1/leaderboards/game1/top?limit=10" | jq .

# Get Player Rank
echo -e "\n--- Get Player3 Rank ---"
curl -s "$BASE_URL/api/v1/leaderboards/game1/player/player3" | jq .

# Get Players Around
echo -e "\n--- Get Players Around Player3 ---"
curl -s "$BASE_URL/api/v1/leaderboards/game1/around/player3?range=2" | jq .

# Get Stats
echo -e "\n--- Get Leaderboard Stats ---"
curl -s "$BASE_URL/api/v1/leaderboards/game1/stats" | jq .

# Test Best Mode - Lower Score
echo -e "\n--- Test Best Mode: Submit Lower Score (should NOT update) ---"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player4", "leaderboard_id": "game1", "score": 2000}' | jq .

echo "Player4 score (should still be 3200):"
curl -s "$BASE_URL/api/v1/leaderboards/game1/player/player4" | jq .

# Test Best Mode - Higher Score
echo -e "\n--- Test Best Mode: Submit Higher Score (should update) ---"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player4", "leaderboard_id": "game1", "score": 5000}' | jq .

echo "Player4 score (should be 5000):"
curl -s "$BASE_URL/api/v1/leaderboards/game1/player/player4" | jq .

# List Leaderboards
echo -e "\n--- List All Leaderboards ---"
curl -s "$BASE_URL/api/v1/leaderboards" | jq .

# Remove Player
echo -e "\n--- Remove Player5 ---"
curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/game1/player/player5" | jq .

# Error Handling
echo -e "\n--- Error: Get Non-existent Player ---"
curl -s "$BASE_URL/api/v1/leaderboards/game1/player/nonexistent" | jq .

echo -e "\n--- Error: Get Non-existent Leaderboard ---"
curl -s "$BASE_URL/api/v1/leaderboards/nonexistent" | jq .

# Test Increment Mode
echo -e "\n--- Create Leaderboard (game2 - increment mode) ---"
curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "game2", "name": "Game 2 - Increment Mode", "update_mode": "increment"}' | jq .

echo -e "\n--- Test Increment Mode ---"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player1", "leaderboard_id": "game2", "score": 100}' | jq .

curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player1", "leaderboard_id": "game2", "score": 50}' | jq .

echo "Player1 in game2 (should be 150):"
curl -s "$BASE_URL/api/v1/leaderboards/game2/player/player1" | jq .

# Duplicate Prevention
echo -e "\n--- Error: Duplicate Leaderboard ---"
curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "game1", "name": "Duplicate"}' | jq .

# Reset Leaderboard
echo -e "\n--- Reset game2 Leaderboard ---"
curl -s -X POST "$BASE_URL/api/v1/leaderboards/game2/reset" | jq .

echo "game2 after reset (should be empty):"
curl -s "$BASE_URL/api/v1/leaderboards/game2/top?limit=10" | jq .

echo -e "\n=========================================="
echo "All tests completed!"
echo "=========================================="
```

## Manual Test Commands

### Health Check
```bash
curl -s http://localhost:8080/health | jq .
```

### Create Leaderboard
```bash
curl -s -X POST http://localhost:8080/api/v1/leaderboards \
  -H "Content-Type: application/json" \
  -d '{
    "id": "game1",
    "name": "Game 1 Leaderboard",
    "sort_order": "desc",
    "update_mode": "best"
  }' | jq .
```

### Submit Score
```bash
curl -s -X POST http://localhost:8080/api/v1/scores \
  -H "Content-Type: application/json" \
  -d '{
    "player_id": "player1",
    "leaderboard_id": "game1",
    "score": 1500
  }' | jq .
```

### Submit Batch Scores
```bash
curl -s -X POST http://localhost:8080/api/v1/scores/batch \
  -H "Content-Type: application/json" \
  -d '{
    "scores": [
      {"player_id": "player1", "leaderboard_id": "game1", "score": 1000},
      {"player_id": "player2", "leaderboard_id": "game1", "score": 2000}
    ]
  }' | jq .
```

### Get Top Players
```bash
curl -s "http://localhost:8080/api/v1/leaderboards/game1/top?limit=10" | jq .
```

### Get Player Rank
```bash
curl -s http://localhost:8080/api/v1/leaderboards/game1/player/player1 | jq .
```

### Get Players Around a Player
```bash
curl -s "http://localhost:8080/api/v1/leaderboards/game1/around/player1?range=5" | jq .
```

### Get Rank Range
```bash
curl -s "http://localhost:8080/api/v1/leaderboards/game1/range?start=0&end=10" | jq .
```

### Get Leaderboard Stats
```bash
curl -s http://localhost:8080/api/v1/leaderboards/game1/stats | jq .
```

### Remove Player
```bash
curl -s -X DELETE http://localhost:8080/api/v1/leaderboards/game1/player/player1 | jq .
```

### Reset Leaderboard
```bash
curl -s -X POST http://localhost:8080/api/v1/leaderboards/game1/reset | jq .
```

### Delete Leaderboard
```bash
curl -s -X DELETE http://localhost:8080/api/v1/leaderboards/game1 | jq .
```

## Cleanup

```bash
# Stop the Go server (Ctrl+C)

# Stop Docker containers
docker-compose down

# Remove volumes (optional - removes all data)
docker-compose down -v
```

