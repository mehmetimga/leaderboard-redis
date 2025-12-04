#!/bin/bash
# Generate burst of random scores (for stress testing)
# Usage: ./scripts/generate-burst.sh [leaderboard_id] [num_scores]

BASE_URL="${BASE_URL:-http://localhost:8080}"
LEADERBOARD_ID="${1:-game1}"
NUM_SCORES="${2:-50}"

PLAYER_NAMES=(
  "Phoenix" "Shadow" "Thunder" "Storm" "Blaze"
  "Ninja" "Dragon" "Wolf" "Hawk" "Viper"
  "Ghost" "Titan" "Frost" "Cyber" "Nova"
)

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  ⚡ Burst Score Generator${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Leaderboard:${NC} $LEADERBOARD_ID"
echo -e "${YELLOW}Scores to submit:${NC} $NUM_SCORES"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Build batch request
echo -e "${YELLOW}Submitting $NUM_SCORES scores in batch...${NC}"

scores="["
for i in $(seq 1 $NUM_SCORES); do
  NAME_INDEX=$((RANDOM % ${#PLAYER_NAMES[@]}))
  PLAYER_NUM=$((RANDOM % 50 + 1))
  PLAYER_NAME="${PLAYER_NAMES[$NAME_INDEX]}$PLAYER_NUM"
  SCORE=$((RANDOM % 2000 + 500))
  
  if [ $i -gt 1 ]; then
    scores="$scores,"
  fi
  scores="$scores{\"player_id\": \"$PLAYER_NAME\", \"leaderboard_id\": \"$LEADERBOARD_ID\", \"score\": $SCORE}"
done
scores="$scores]"

# Submit batch
response=$(curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
  -H "Content-Type: application/json" \
  -d "{\"scores\": $scores}")

echo -e "${GREEN}✓ Submitted $NUM_SCORES scores${NC}"
echo ""
echo "Response: $response"

