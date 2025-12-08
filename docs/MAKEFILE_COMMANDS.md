# Makefile Commands Guide

This document provides a comprehensive guide to all available Makefile commands for the Leaderboard System.

## Quick Commands Reference

```bash
# Start everything
make run

# Feed data
make feed-burst LEADERBOARD=game1 PLAYERS=500

# Check status
make status
make test-health

# View data
make test-leaderboard LEADERBOARD=game1

# Stop
make stop
```

## Quick Start

```bash
# Start everything (Docker + Server + Webapp)
make run

# Feed test data via Kafka
make feed

# Open webapp in browser
open http://localhost:3000

# Stop everything when done
make stop
```

## Command Reference

### 🔨 Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build all binaries (server + kafka-producer) |
| `make build-server` | Build server binary only |
| `make build-producer` | Build kafka-producer binary only |

```bash
# Build everything
make build

# Output:
# ✅ Server built: bin/server
# ✅ Kafka producer built: bin/kafka-producer
```

---

### 🚀 Run Commands

| Command | Description |
|---------|-------------|
| `make run` | Start everything (docker + server + webapp) |
| `make run-docker` | Start Docker services (Redis, PostgreSQL, Kafka) |
| `make run-server` | Start the Go server |
| `make run-webapp` | Start the React webapp |

```bash
# Start all services
make run

# Output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#   🚀 All services started!
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#   Server:  http://localhost:8080
#   WebApp:  http://localhost:3000
#   Kafka:   localhost:9094
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Start only Docker services
make run-docker

# Start only the server (after Docker is running)
make run-server

# Start only the webapp
make run-webapp
```

---

### 🛑 Stop Commands

| Command | Description |
|---------|-------------|
| `make stop` | Stop everything |
| `make stop-docker` | Stop Docker services only |
| `make stop-server` | Stop the Go server only |
| `make stop-webapp` | Stop the webapp only |

```bash
# Stop all services
make stop

# Output:
# Stopping server...
# ✅ Server stopped
# Stopping webapp...
# ✅ Webapp stopped
# Stopping Docker services...
# ✅ Docker services stopped
# ✅ All services stopped
```

---

### 📊 Feed Commands (Data Ingestion)

| Command | Description |
|---------|-------------|
| `make feed` | Feed 1000 players via Kafka (100/sec) |
| `make feed-kafka` | Feed via Kafka (customizable) |
| `make feed-http` | Feed via HTTP API (using script) |
| `make feed-burst` | Send burst of data via Kafka |
| `make feed-battle` | Run battle royale demo |

#### Default Feed
```bash
# Feed 1000 players at 100 updates/sec
make feed
```

#### Customizable Kafka Feed
```bash
# Custom parameters
make feed-kafka LEADERBOARD=game1 PLAYERS=500 RATE=50

# With duration limit (30 seconds)
make feed-kafka LEADERBOARD=game1 PLAYERS=1000 RATE=200 DURATION=30
```

**Available Parameters:**
| Parameter | Default | Description |
|-----------|---------|-------------|
| `LEADERBOARD` | `game1` | Target leaderboard ID |
| `PLAYERS` | `1000` | Number of players to create |
| `RATE` | `100` | Updates per second |
| `DURATION` | `0` | Duration in seconds (0 = forever) |

#### Burst Mode (Fast Initial Load)
```bash
# Quickly load 5000 players (no continuous updates)
make feed-burst PLAYERS=5000 LEADERBOARD=game1
```

#### HTTP Feed (Alternative)
```bash
# Feed via HTTP API instead of Kafka
make feed-http LEADERBOARD=game1
```

---

### 🧪 Test Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-health` | Check service health |
| `make test-api` | Test HTTP API endpoints |
| `make test-leaderboard` | Show leaderboard data |

#### Health Check
```bash
make test-health

# Output:
# Testing service health...
#   Server:   ✅ Healthy
#   Redis:    ✅ Healthy
#   Postgres: ✅ Healthy
#   Kafka:    ✅ Healthy
```

#### API Test
```bash
make test-api

# Output:
#   GET /health
#   { "success": true, "data": { "status": "healthy" } }
#
#   GET /api/v1/leaderboards
#   Found 3 leaderboards
```

#### View Leaderboard
```bash
make test-leaderboard LEADERBOARD=game1

# Output:
# Leaderboard: game1
#
# Stats:
# { "total_players": 1000, "top_score": 50000 }
#
# Top 10:
# [{ "rank": 1, "player_id": "Player1", "score": 50000 }, ...]
```

#### Create Leaderboard
```bash
make create-leaderboard LEADERBOARD=my-game

# Creates a new leaderboard with increment mode
```

---

### 📋 Logs & Status

| Command | Description |
|---------|-------------|
| `make status` | Show status of all services |
| `make logs` | Show server logs |
| `make logs-webapp` | Show webapp logs |
| `make logs-docker` | Show Docker container logs |
| `make logs-kafka` | Show Kafka logs |
| `make watch-logs` | Watch server logs in real-time |

```bash
# Check status of all services
make status

# Output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#   Service Status
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Docker Containers:
# NAME                 STATUS         PORTS
# leaderboard-kafka    Up (healthy)   9092, 9094
# leaderboard-redis    Up (healthy)   6379
# leaderboard-postgres Up (healthy)   5433
#
# Go Server:
#   ✅ Running (PID: 12345)
#
# Webapp:
#   ✅ Running (PID: 12346)

# View recent server logs
make logs

# Watch logs in real-time
make watch-logs
```

---

### 🧹 Clean Commands

| Command | Description |
|---------|-------------|
| `make clean` | Clean build artifacts |
| `make clean-docker` | Remove Docker volumes |
| `make clean-all` | Full cleanup (artifacts + volumes) |

```bash
# Clean build artifacts
make clean

# Full cleanup including Docker data
make clean-all
```

---

### 🔄 Development Helpers

| Command | Description |
|---------|-------------|
| `make restart` | Quick restart all services |
| `make restart-server` | Restart server only |
| `make open-webapp` | Open webapp in browser (macOS) |

```bash
# Quick restart after code changes
make restart-server

# Full restart
make restart
```

---

## Common Workflows

### 1. Development Setup
```bash
# Initial setup
make run

# Create a test leaderboard
make create-leaderboard LEADERBOARD=dev-test

# Feed some test data
make feed-burst LEADERBOARD=dev-test PLAYERS=100

# View the leaderboard
make test-leaderboard LEADERBOARD=dev-test
```

### 2. Load Testing
```bash
# Start services
make run-docker
make run-server

# Run high-load test (5000 players, 500/sec for 60s)
make feed-kafka LEADERBOARD=load-test PLAYERS=5000 RATE=500 DURATION=60

# Monitor logs
make watch-logs
```

### 3. Demo Mode
```bash
# Start everything
make run

# Run battle royale demo
make feed-battle

# Open webapp to watch
make open-webapp
```

### 4. Cleanup
```bash
# Stop services
make stop

# Full cleanup (remove all data)
make clean-all
```

---

## Architecture Reference

```
┌─────────────────────────────────────────────────────────────────┐
│                     make feed / make feed-kafka                 │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────┐      ┌─────────────┐      ┌────────────────┐  │
│  │   Kafka     │─────►│  Consumer   │─────►│  Leaderboard   │  │
│  │  Producer   │      │  (Server)   │      │    Service     │  │
│  └─────────────┘      └─────────────┘      └───────┬────────┘  │
│                                                    │            │
│                              ┌─────────────────────┴──────┐     │
│                              ▼                            ▼     │
│                       ┌───────────┐               ┌───────────┐ │
│                       │   Redis   │               │ PostgreSQL│ │
│                       └───────────┘               └───────────┘ │
│                              │                                  │
│                              ▼                                  │
│                       ┌───────────┐                             │
│                       │ WebSocket │◄──── make run-webapp        │
│                       │    Hub    │                             │
│                       └───────────┘                             │
│                              │                                  │
│                              ▼                                  │
│                       ┌───────────┐                             │
│                       │  Browser  │  http://localhost:3000      │
│                       └───────────┘                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Troubleshooting

### Services won't start
```bash
# Check what's running
make status

# Check Docker logs
make logs-docker

# Try stopping and restarting
make stop
make run
```

### Kafka connection issues
```bash
# Verify Kafka is healthy
make test-health

# Check Kafka logs
make logs-kafka

# Restart Docker services
make stop-docker
make run-docker
```

### Server crashes
```bash
# Check server logs
make logs

# Restart server only
make restart-server
```
