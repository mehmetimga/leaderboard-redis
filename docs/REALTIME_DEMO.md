# Real-time Leaderboard Demo Guide

This guide explains how to test the real-time leaderboard system with 1000 players and live updates.

## Architecture Overview

```
┌─────────────────┐     WebSocket      ┌─────────────────┐
│   React App     │◄──────────────────►│   Go Server     │
│   (port 3000)   │                    │   (port 8080)   │
└─────────────────┘                    └────────┬────────┘
                                                │
                           ┌────────────────────┼────────────────────┐
                           │                    │                    │
                           ▼                    ▼                    ▼
                    ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
                    │    Redis    │      │  PostgreSQL │      │ Feed Script │
                    │  (Sorted    │      │  (Source of │      │ (Generates  │
                    │   Sets)     │      │   Truth)    │      │   Scores)   │
                    └─────────────┘      └─────────────┘      └─────────────┘
```

## Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- `jq` and `bc` (for scripts)

## Quick Start

### 1. Start Infrastructure

```bash
cd /path/to/leaderboard-redis

# Start Redis and PostgreSQL
docker-compose up -d redis postgres

# Verify services are healthy
docker-compose ps
```

Expected output:
```
NAME                   STATUS
leaderboard-postgres   Up (healthy)
leaderboard-redis      Up (healthy)
```

### 2. Start Backend Server

```bash
# In terminal 1
go run ./cmd/server -config config.yaml
```

Expected output:
```json
{"level":"INFO","msg":"connecting to Redis","addr":"localhost:6379"}
{"level":"INFO","msg":"connected to Redis"}
{"level":"INFO","msg":"connecting to PostgreSQL"}
{"level":"INFO","msg":"connected to PostgreSQL"}
{"level":"INFO","msg":"WebSocket hub initialized"}
{"level":"INFO","msg":"starting HTTP server","port":8080}
{"level":"INFO","msg":"WebSocket endpoint available at /ws"}
```

### 3. Start Frontend

```bash
# In terminal 2
cd webapp
npm install
npm run dev -- --port 3000
```

Expected output:
```
VITE ready in 152 ms
➜  Local:   http://localhost:3000/
```

### 4. Open Browser

Navigate to: **http://localhost:3000**

---

## Feed Scripts

### Option A: Battle Royale (Dramatic Changes)

Best for demonstrating real-time updates with constant rank changes.

```bash
# In terminal 3
./scripts/battle-royale.sh
```

**Features:**
- Creates **1000 players** with scores 5000-10000
- Updates **10 players/second**
- Uses **REPLACE mode** (scores change completely)
- High variance (6000-12000 point scores)
- **Top 10 changes constantly!**

**What you'll see:**
- Rankings shift every second
- New players jump into top 10
- Leaders get dethroned frequently

### Option B: Standard Feed (Stable Top)

Best for demonstrating "best score" mode where top players stay stable.

```bash
# In terminal 3
./scripts/feed-leaderboard.sh
```

**Features:**
- Creates **1000 players** with scores 1000-6000
- Updates **10 players/second**
- Uses **BEST mode** (only higher scores update)
- 70% bias toward top players
- Top players stay stable, lower ranks change

---

## Step-by-Step Demo

### Demo 1: Battle Royale (Recommended)

1. **Create the leaderboard:**
```bash
curl -X POST http://localhost:8080/api/v1/leaderboards \
  -H "Content-Type: application/json" \
  -d '{"id": "battle-royale", "name": "⚔️ Battle Royale", "update_mode": "replace"}'
```

2. **Open browser** at http://localhost:3000

3. **Click "⚔️ Battle Royale"** button

4. **Start the feed:**
```bash
./scripts/battle-royale.sh battle-royale
```

5. **Watch the leaderboard change in real-time!**
   - Top 3 podium updates constantly
   - Rank change arrows (▼) appear
   - Player count shows 1000
   - "Live" indicator pulses green

### Demo 2: 1000 Player Top 10

1. **Create the leaderboard:**
```bash
curl -X POST http://localhost:8080/api/v1/leaderboards \
  -H "Content-Type: application/json" \
  -d '{"id": "top10-test", "name": "🏆 Top 10 Battle", "update_mode": "best"}'
```

2. **Start the feed:**
```bash
./scripts/feed-leaderboard.sh top10-test
```

3. **Watch in browser** - top scores stay stable, lower ranks shuffle

---

## Script Reference

### battle-royale.sh

```bash
./scripts/battle-royale.sh [leaderboard_id]
```

| Parameter | Default | Description |
|-----------|---------|-------------|
| leaderboard_id | battle-royale | Target leaderboard |

**Output:**
```
⚔️  BATTLE ROYALE - 1000 Players
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Mode: REPLACE (scores change constantly!)
Updates: 10 players/second with high variance

Creating 1000 players...
  Progress: 1000/1000

Initial Top 10:
  # 1 Frost16         9,998 pts
  # 2 Raven16         9,993 pts
  ...

⚔️  Starting Battle! Scores changing constantly...

[11:11:34] ⚔️  Batch #5 | 50 score updates
[11:11:40] ⚔️  Batch #10 | 100 score updates
```

### feed-leaderboard.sh

```bash
./scripts/feed-leaderboard.sh [leaderboard_id]
```

| Parameter | Default | Description |
|-----------|---------|-------------|
| leaderboard_id | realtime-test | Target leaderboard |

**Output:**
```
🏆 Leaderboard Feed System - 1000 Players
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Players: 1000
Updates/sec: 10

Creating 1000 players...
Starting continuous updates (10 players/second)
Top players have 70% chance to be updated
```

### generate-burst.sh

For quick stress tests:

```bash
./scripts/generate-burst.sh [leaderboard_id] [num_scores]
```

Example:
```bash
# Send 100 scores instantly
./scripts/generate-burst.sh battle-royale 100
```

---

## WebSocket Protocol

### Connecting

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

### Subscribe to Leaderboard

```json
{
  "type": "subscribe",
  "leaderboard_id": "battle-royale"
}
```

### Receive Updates

```json
{
  "type": "leaderboard_update",
  "leaderboard_id": "battle-royale",
  "data": {
    "leaderboard_id": "battle-royale",
    "entries": [
      {"rank": 1, "player_id": "Phoenix1", "score": 11989},
      {"rank": 2, "player_id": "Alpha5", "score": 11983},
      ...
    ],
    "total_players": 1000
  },
  "timestamp": "2025-12-04T11:12:00Z"
}
```

---

## Troubleshooting

### Port Already in Use

```bash
# Kill process on port 8080
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Kill process on port 3000
lsof -i :3000 | grep LISTEN | awk '{print $2}' | xargs kill -9
```

### Redis/PostgreSQL Not Running

```bash
docker-compose up -d redis postgres
docker-compose ps  # Check status
```

### WebSocket Disconnecting

Check browser console for errors. The frontend auto-reconnects with exponential backoff.

### Feed Script Errors

If you see `declare: -A: invalid option`, the script still works (it's a bash vs zsh difference).

---

## API Endpoints Used

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/leaderboards` | Create leaderboard |
| POST | `/api/v1/leaderboards/{id}/reset` | Reset scores |
| POST | `/api/v1/scores/batch` | Submit batch scores |
| GET | `/api/v1/leaderboards/{id}/top?limit=10` | Get top 10 |
| GET | `/ws` | WebSocket connection |

---

## Stopping Everything

```bash
# Stop feed script
pkill -f "battle-royale.sh"
pkill -f "feed-leaderboard.sh"

# Stop servers (Ctrl+C in their terminals)

# Stop Docker services
docker-compose down
```

---

## Performance Notes

- **1000 players**: Creates in ~10 seconds
- **10 updates/second**: ~600 updates/minute
- **WebSocket latency**: <50ms typical
- **Redis sorted sets**: O(log N) operations
- **Top 10 query**: <1ms

---

## Screenshots

The demo produces dramatic rank changes:

| Time | #1 | #2 | #3 | #4 | #5 |
|------|----|----|----|----|-----|
| 0s | Storm10 | Ghost18 | Delta5 | Raven19 | Blaze2 |
| 5s | ??? | Storm10 | Ghost18 | Haze1 | Crash1 |
| 10s | ??? | Alpha5 | Nova1 | Storm10 | Ghost18 |

Almost complete top 10 turnover in 10 seconds! 🏆

