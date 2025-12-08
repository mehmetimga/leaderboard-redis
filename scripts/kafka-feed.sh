#!/bin/bash
# Feed leaderboard data through Kafka
# Usage: ./scripts/kafka-feed.sh [leaderboard_id] [players] [rate]

LEADERBOARD_ID="${1:-game1}"
TOTAL_PLAYERS="${2:-1000}"
UPDATES_PER_SECOND="${3:-100}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9094}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  🚀 Kafka Leaderboard Feeder${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Leaderboard:${NC} $LEADERBOARD_ID"
echo -e "${YELLOW}Players:${NC} $TOTAL_PLAYERS"
echo -e "${YELLOW}Rate:${NC} $UPDATES_PER_SECOND updates/sec"
echo -e "${YELLOW}Brokers:${NC} $KAFKA_BROKERS"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Build the producer if not exists
PRODUCER_BIN="./bin/kafka-producer"
if [ ! -f "$PRODUCER_BIN" ]; then
    echo -e "${YELLOW}Building kafka-producer...${NC}"
    go build -o "$PRODUCER_BIN" ./cmd/kafka-producer
    if [ $? -ne 0 ]; then
        echo -e "${GREEN}Build failed. Running with go run instead...${NC}"
        go run ./cmd/kafka-producer \
            -brokers "$KAFKA_BROKERS" \
            -leaderboard "$LEADERBOARD_ID" \
            -players "$TOTAL_PLAYERS" \
            -rate "$UPDATES_PER_SECOND"
        exit $?
    fi
    echo -e "${GREEN}✓ Built successfully${NC}"
fi

# Run the producer
"$PRODUCER_BIN" \
    -brokers "$KAFKA_BROKERS" \
    -leaderboard "$LEADERBOARD_ID" \
    -players "$TOTAL_PLAYERS" \
    -rate "$UPDATES_PER_SECOND"
