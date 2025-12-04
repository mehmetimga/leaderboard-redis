#!/bin/bash
# Feed leaderboard with 1000 users and continuous updates
# Biased toward top players to show leaderboard changes
# Usage: ./scripts/feed-leaderboard.sh [leaderboard_id]

BASE_URL="${BASE_URL:-http://localhost:8080}"
LEADERBOARD_ID="${1:-realtime-test}"
TOTAL_PLAYERS=1000
UPDATES_PER_SECOND=10

# Player name prefixes for variety
PREFIXES=(
  "Phoenix" "Shadow" "Thunder" "Storm" "Blaze" "Ninja" "Dragon" "Wolf" "Hawk" "Viper"
  "Ghost" "Titan" "Frost" "Cyber" "Nova" "Raven" "Omega" "Alpha" "Delta" "Sigma"
  "Ace" "Bolt" "Crash" "Dash" "Edge" "Flash" "Glitch" "Haze" "Ion" "Jade"
  "Knight" "Luna" "Mystic" "Neon" "Orion" "Pulse" "Quantum" "Rebel" "Spark" "Turbo"
)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

clear
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  🏆 Leaderboard Feed System - 1000 Players${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Leaderboard:${NC} $LEADERBOARD_ID"
echo -e "${YELLOW}Total Players:${NC} $TOTAL_PLAYERS"
echo -e "${YELLOW}Updates/sec:${NC} $UPDATES_PER_SECOND"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Reset the leaderboard first
echo -e "${YELLOW}Resetting leaderboard...${NC}"
curl -s -X POST "$BASE_URL/api/v1/leaderboards/$LEADERBOARD_ID/reset" > /dev/null 2>&1

# Generate player name
get_player_name() {
  local idx=$1
  local prefix_idx=$((idx % ${#PREFIXES[@]}))
  local suffix=$((idx / ${#PREFIXES[@]} + 1))
  echo "${PREFIXES[$prefix_idx]}${suffix}"
}

# Build batch of scores
build_batch() {
  local start=$1
  local end=$2
  local scores="["
  local first=true
  
  for i in $(seq $start $end); do
    local player=$(get_player_name $i)
    local score=$((RANDOM % 5000 + 1000))
    
    if [ "$first" = true ]; then
      first=false
    else
      scores="$scores,"
    fi
    scores="$scores{\"player_id\":\"$player\",\"leaderboard_id\":\"$LEADERBOARD_ID\",\"score\":$score}"
  done
  scores="$scores]"
  echo "$scores"
}

# Create 1000 players in batches
echo -e "${CYAN}Creating $TOTAL_PLAYERS players...${NC}"
BATCH_SIZE=100
for batch_start in $(seq 0 $BATCH_SIZE $((TOTAL_PLAYERS - 1))); do
  batch_end=$((batch_start + BATCH_SIZE - 1))
  if [ $batch_end -ge $TOTAL_PLAYERS ]; then
    batch_end=$((TOTAL_PLAYERS - 1))
  fi
  
  scores=$(build_batch $batch_start $batch_end)
  curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
    -H "Content-Type: application/json" \
    -d "{\"scores\":$scores}" > /dev/null
  
  progress=$((batch_end * 100 / TOTAL_PLAYERS))
  echo -ne "\r  Progress: ${GREEN}$((batch_end + 1))${NC}/$TOTAL_PLAYERS players (${progress}%)"
done

echo -e "\n${GREEN}✓ Created $TOTAL_PLAYERS players${NC}"
echo ""

# Get initial top 10
echo -e "${CYAN}Initial Top 10:${NC}"
curl -s "$BASE_URL/api/v1/leaderboards/$LEADERBOARD_ID/top?limit=10" | \
  python3 -c "
import sys, json
data = json.load(sys.stdin)
if data.get('success') and data.get('data'):
    for e in data['data']:
        print(f\"  #{e['rank']:2} {e['player_id']:15} {e['score']:,} pts\")
"

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Starting continuous updates (10 players/second)${NC}"
echo -e "${YELLOW}Top players have 70% chance to be updated (to create movement)${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Track update counts
declare -A update_counts
update_num=0

while true; do
  # Build batch of 10 updates with bias toward top players
  scores="["
  first=true
  
  for i in $(seq 1 $UPDATES_PER_SECOND); do
    # 70% chance to pick from top 20 players, 30% from rest
    if [ $((RANDOM % 100)) -lt 70 ]; then
      # Pick from top 20 (indices 0-19)
      player_idx=$((RANDOM % 20))
    else
      # Pick from remaining players (20-999)
      player_idx=$((RANDOM % 980 + 20))
    fi
    
    player=$(get_player_name $player_idx)
    
    # Score boost: larger for top players attempting to stay on top
    if [ $player_idx -lt 10 ]; then
      score=$((RANDOM % 800 + 400))  # 400-1200 for top 10
    elif [ $player_idx -lt 50 ]; then
      score=$((RANDOM % 600 + 300))  # 300-900 for top 50
    else
      score=$((RANDOM % 400 + 200))  # 200-600 for others
    fi
    
    if [ "$first" = true ]; then
      first=false
    else
      scores="$scores,"
    fi
    scores="$scores{\"player_id\":\"$player\",\"leaderboard_id\":\"$LEADERBOARD_ID\",\"score\":$score}"
    
    # Track updates
    update_counts[$player]=$((${update_counts[$player]:-0} + 1))
  done
  scores="$scores]"
  
  # Submit batch
  curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
    -H "Content-Type: application/json" \
    -d "{\"scores\":$scores}" > /dev/null
  
  update_num=$((update_num + 1))
  timestamp=$(date +"%H:%M:%S")
  
  # Show update info every 5 batches
  if [ $((update_num % 5)) -eq 0 ]; then
    total_updates=$((update_num * UPDATES_PER_SECOND))
    echo -e "[$timestamp] ${GREEN}Batch #$update_num${NC} | Total updates: ${CYAN}$total_updates${NC}"
  fi
  
  # Sleep for 1 second (10 updates per second means 1 batch per second)
  sleep 1
done

