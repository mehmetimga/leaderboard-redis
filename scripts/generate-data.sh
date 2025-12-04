#!/bin/bash
# Generate random leaderboard data
# Usage: ./scripts/generate-data.sh [leaderboard_id] [interval_ms] [num_players]

BASE_URL="${BASE_URL:-http://localhost:8080}"
LEADERBOARD_ID="${1:-game1}"
INTERVAL_MS="${2:-1000}"
NUM_PLAYERS="${3:-20}"

# Player names for more realistic data
PLAYER_NAMES=(
  "Phoenix" "Shadow" "Thunder" "Storm" "Blaze"
  "Ninja" "Dragon" "Wolf" "Hawk" "Viper"
  "Ghost" "Titan" "Frost" "Cyber" "Nova"
  "Raven" "Omega" "Alpha" "Delta" "Sigma"
  "Ace" "Bolt" "Crash" "Dash" "Edge"
  "Flash" "Glitch" "Haze" "Ion" "Jade"
)

# Colors for terminal output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  🎮 Leaderboard Data Generator${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Leaderboard:${NC} $LEADERBOARD_ID"
echo -e "${YELLOW}Interval:${NC} ${INTERVAL_MS}ms"
echo -e "${YELLOW}Players:${NC} $NUM_PLAYERS"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Create initial players if leaderboard is empty
echo -e "${YELLOW}Creating initial players...${NC}"
for i in $(seq 1 $NUM_PLAYERS); do
  NAME_INDEX=$((RANDOM % ${#PLAYER_NAMES[@]}))
  PLAYER_NAME="${PLAYER_NAMES[$NAME_INDEX]}$i"
  INITIAL_SCORE=$((RANDOM % 5000 + 1000))
  
  curl -s -X POST "$BASE_URL/api/v1/scores" \
    -H "Content-Type: application/json" \
    -d "{\"player_id\": \"$PLAYER_NAME\", \"leaderboard_id\": \"$LEADERBOARD_ID\", \"score\": $INITIAL_SCORE}" > /dev/null
  
  echo -e "  Created ${GREEN}$PLAYER_NAME${NC} with score ${GREEN}$INITIAL_SCORE${NC}"
done

echo ""
echo -e "${YELLOW}Starting random score updates...${NC}"
echo ""

count=0
while true; do
  # Pick a random player
  PLAYER_INDEX=$((RANDOM % NUM_PLAYERS + 1))
  NAME_INDEX=$(( (PLAYER_INDEX - 1) % ${#PLAYER_NAMES[@]} ))
  PLAYER_NAME="${PLAYER_NAMES[$NAME_INDEX]}$PLAYER_INDEX"
  
  # Generate random score (can be positive or add to existing)
  SCORE=$((RANDOM % 500 + 100))
  
  # Submit score
  response=$(curl -s -X POST "$BASE_URL/api/v1/scores" \
    -H "Content-Type: application/json" \
    -d "{\"player_id\": \"$PLAYER_NAME\", \"leaderboard_id\": \"$LEADERBOARD_ID\", \"score\": $SCORE}")
  
  count=$((count + 1))
  timestamp=$(date +"%H:%M:%S")
  
  echo -e "[$timestamp] #$count ${GREEN}$PLAYER_NAME${NC} +${YELLOW}$SCORE${NC} pts"
  
  # Sleep for the specified interval (convert ms to seconds)
  sleep $(echo "scale=3; $INTERVAL_MS/1000" | bc)
done

