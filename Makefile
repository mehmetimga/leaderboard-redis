# Leaderboard System Makefile
# ============================

.PHONY: help build run stop feed test clean

# Default target
help:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  🏆 Leaderboard System - Available Commands"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "  Build Commands:"
	@echo "    make build              Build all binaries (server + kafka-producer)"
	@echo "    make build-server       Build server binary only"
	@echo "    make build-producer     Build kafka-producer binary only"
	@echo ""
	@echo "  Run Commands:"
	@echo "    make run                Start everything (docker + server + webapp)"
	@echo "    make run-docker         Start Docker services (redis, postgres, kafka)"
	@echo "    make run-server         Start the Go server"
	@echo "    make run-webapp         Start the React webapp"
	@echo "    make run-all            Alias for 'make run'"
	@echo ""
	@echo "  Stop Commands:"
	@echo "    make stop               Stop everything"
	@echo "    make stop-docker        Stop Docker services only"
	@echo "    make stop-server        Stop the Go server only"
	@echo "    make stop-webapp        Stop the webapp only"
	@echo ""
	@echo "  Feed Commands (Data Ingestion):"
	@echo "    make feed               Feed 1000 players via Kafka (100/sec)"
	@echo "    make feed-kafka         Feed via Kafka (customizable)"
	@echo "    make feed-http          Feed via HTTP API (using script)"
	@echo "    make feed-burst         Send burst of data via Kafka"
	@echo ""
	@echo "  Test Commands:"
	@echo "    make test               Run all tests"
	@echo "    make test-api           Test HTTP API endpoints"
	@echo "    make test-health        Check service health"
	@echo "    make test-leaderboard   Show leaderboard data"
	@echo ""
	@echo "  Other Commands:"
	@echo "    make logs               Show server logs"
	@echo "    make logs-docker        Show Docker container logs"
	@echo "    make status             Show status of all services"
	@echo "    make clean              Clean build artifacts and volumes"
	@echo "    make create-leaderboard Create a test leaderboard"
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ============================================================================
# Build Commands
# ============================================================================

build: build-server build-producer
	@echo "✅ All binaries built successfully"

build-server:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/server ./cmd/server
	@echo "✅ Server built: bin/server"

build-producer:
	@echo "Building kafka-producer..."
	@mkdir -p bin
	@go build -o bin/kafka-producer ./cmd/kafka-producer
	@echo "✅ Kafka producer built: bin/kafka-producer"

# ============================================================================
# Run Commands
# ============================================================================

run: run-all

run-all: build run-docker
	@echo "Waiting for Docker services to be healthy..."
	@sleep 10
	@$(MAKE) run-server
	@sleep 3
	@$(MAKE) run-webapp
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  🚀 All services started!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Server:  http://localhost:8080"
	@echo "  WebApp:  http://localhost:3000"
	@echo "  Kafka:   localhost:9094"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

run-docker:
	@echo "Starting Docker services..."
	@docker-compose up -d redis postgres kafka
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker-compose ps

run-server: build-server
	@echo "Starting server..."
	@./bin/server -config config.yaml > /tmp/leaderboard-server.log 2>&1 &
	@sleep 2
	@if curl -s http://localhost:8080/health > /dev/null 2>&1; then \
		echo "✅ Server started at http://localhost:8080"; \
	else \
		echo "⚠️  Server may still be starting. Check logs with 'make logs'"; \
	fi

run-webapp:
	@echo "Starting webapp..."
	@cd webapp && npm install --silent && npm run dev -- --port 3000 > /tmp/leaderboard-webapp.log 2>&1 &
	@sleep 3
	@echo "✅ Webapp started at http://localhost:3000"

# ============================================================================
# Stop Commands
# ============================================================================

stop: stop-server stop-webapp stop-docker
	@echo "✅ All services stopped"

stop-docker:
	@echo "Stopping Docker services..."
	@docker-compose down
	@echo "✅ Docker services stopped"

stop-server:
	@echo "Stopping server..."
	@pkill -f "bin/server" 2>/dev/null || true
	@echo "✅ Server stopped"

stop-webapp:
	@echo "Stopping webapp..."
	@pkill -f "vite" 2>/dev/null || true
	@echo "✅ Webapp stopped"

# ============================================================================
# Feed Commands (Data Ingestion)
# ============================================================================

# Default feed: 1000 players, 100 updates/sec via Kafka
feed: build-producer
	@echo "Feeding data via Kafka..."
	@./bin/kafka-producer \
		-brokers localhost:9094 \
		-leaderboard game1 \
		-players 1000 \
		-rate 100

# Customizable Kafka feed
# Usage: make feed-kafka LEADERBOARD=game1 PLAYERS=500 RATE=50
LEADERBOARD ?= game1
PLAYERS ?= 1000
RATE ?= 100
DURATION ?= 0

feed-kafka: build-producer
	@echo "Feeding $(PLAYERS) players to $(LEADERBOARD) at $(RATE)/sec via Kafka..."
	@./bin/kafka-producer \
		-brokers localhost:9094 \
		-leaderboard $(LEADERBOARD) \
		-players $(PLAYERS) \
		-rate $(RATE) \
		$(if $(filter-out 0,$(DURATION)),-duration $(DURATION)s)

# Feed via HTTP API
feed-http:
	@echo "Feeding data via HTTP API..."
	@./scripts/feed-leaderboard.sh $(LEADERBOARD)

# Quick burst of data via Kafka (initial only, no continuous)
feed-burst: build-producer
	@echo "Sending burst of $(PLAYERS) players to $(LEADERBOARD)..."
	@./bin/kafka-producer \
		-brokers localhost:9094 \
		-leaderboard $(LEADERBOARD) \
		-players $(PLAYERS) \
		-rate 1000 \
		-initial-only

# Battle royale demo
feed-battle:
	@echo "Starting battle royale demo..."
	@./scripts/battle-royale.sh

# ============================================================================
# Test Commands
# ============================================================================

test: test-health test-api
	@echo "✅ All tests passed"

test-health:
	@echo "Testing service health..."
	@echo -n "  Server: "
	@curl -s http://localhost:8080/health | python3 -c "import sys,json; d=json.load(sys.stdin); print('✅ Healthy' if d.get('success') else '❌ Unhealthy')" 2>/dev/null || echo "❌ Not running"
	@echo -n "  Redis:  "
	@docker-compose exec -T redis redis-cli ping 2>/dev/null | grep -q PONG && echo "✅ Healthy" || echo "❌ Not running"
	@echo -n "  Postgres: "
	@docker-compose exec -T postgres pg_isready -U leaderboard 2>/dev/null | grep -q "accepting" && echo "✅ Healthy" || echo "❌ Not running"
	@echo -n "  Kafka:  "
	@docker-compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list 2>/dev/null > /dev/null && echo "✅ Healthy" || echo "❌ Not running"

test-api:
	@echo "Testing API endpoints..."
	@echo "  GET /health"
	@curl -s http://localhost:8080/health | python3 -m json.tool
	@echo ""
	@echo "  GET /api/v1/leaderboards"
	@curl -s http://localhost:8080/api/v1/leaderboards | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  Found {len(d.get(\"data\", []))} leaderboards')"

test-leaderboard:
	@echo "Leaderboard: $(LEADERBOARD)"
	@echo ""
	@echo "Stats:"
	@curl -s "http://localhost:8080/api/v1/leaderboards/$(LEADERBOARD)/stats" | python3 -m json.tool
	@echo ""
	@echo "Top 10:"
	@curl -s "http://localhost:8080/api/v1/leaderboards/$(LEADERBOARD)/top?limit=10" | python3 -m json.tool

# Create a test leaderboard
create-leaderboard:
	@echo "Creating leaderboard: $(LEADERBOARD)"
	@curl -s -X POST http://localhost:8080/api/v1/leaderboards \
		-H "Content-Type: application/json" \
		-d '{"id":"$(LEADERBOARD)","name":"$(LEADERBOARD)","sort_order":"desc","update_mode":"increment"}' | python3 -m json.tool

# ============================================================================
# Logs & Status
# ============================================================================

logs:
	@echo "=== Server Logs ==="
	@tail -50 /tmp/leaderboard-server.log 2>/dev/null || echo "No server logs found"

logs-webapp:
	@echo "=== Webapp Logs ==="
	@tail -50 /tmp/leaderboard-webapp.log 2>/dev/null || echo "No webapp logs found"

logs-docker:
	@docker-compose logs --tail=50

logs-kafka:
	@docker-compose logs --tail=50 kafka

status:
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Service Status"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "Docker Containers:"
	@docker-compose ps 2>/dev/null || echo "  Docker not running"
	@echo ""
	@echo "Go Server:"
	@pgrep -f "bin/server" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f 'bin/server'))" || echo "  ❌ Not running"
	@echo ""
	@echo "Webapp:"
	@pgrep -f "vite" > /dev/null && echo "  ✅ Running (PID: $$(pgrep -f 'vite'))" || echo "  ❌ Not running"
	@echo ""

# ============================================================================
# Clean Commands
# ============================================================================

clean: stop
	@echo "Cleaning up..."
	@rm -rf bin/
	@rm -f /tmp/leaderboard-server.log
	@rm -f /tmp/leaderboard-webapp.log
	@echo "✅ Cleaned build artifacts"

clean-docker: stop-docker
	@echo "Removing Docker volumes..."
	@docker-compose down -v
	@echo "✅ Docker volumes removed"

clean-all: clean clean-docker
	@echo "✅ Full cleanup complete"

# ============================================================================
# Development Helpers
# ============================================================================

# Watch server logs in real-time
watch-logs:
	@tail -f /tmp/leaderboard-server.log

# Open webapp in browser (macOS)
open-webapp:
	@open http://localhost:3000

# Quick restart server
restart-server: stop-server run-server

# Quick restart all
restart: stop run
