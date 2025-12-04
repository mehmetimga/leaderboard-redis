#!/bin/bash
# Battle Royale - Dynamic leaderboard with constant changes
# Shows top 10 changing frequently as scores are replaced
# Usage: ./scripts/battle-royale.sh [leaderboard_id]

BASE_URL="${BASE_URL:-http://localhost:8080}"
LEADERBOARD_ID="${1:-battle-royale}"
TOTAL_PLAYERS=1000

PREFIXES=(
  "Phoenix" "Shadow" "Thunder" "Storm" "Blaze" "Ninja" "Dragon" "Wolf" "Hawk" "Viper"
  "Ghost" "Titan" "Frost" "Cyber" "Nova" "Raven" "Omega" "Alpha" "Delta" "Sigma"
  "Ace" "Bolt" "Crash" "Dash" "Edge" "Flash" "Glitch" "Haze" "Ion" "Jade"
  "Knight" "Luna" "Mystic" "Neon" "Orion" "Pulse" "Quantum" "Rebel" "Spark" "Turbo"
)

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

clear
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${MAGENTA}  ⚔️  BATTLE ROYALE - 1000 Players${NC}"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Mode:${NC} REPLACE (scores change constantly!)"
echo -e "${YELLOW}Updates:${NC} 10 players/second with high variance"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Reset
echo -e "${YELLOW}Resetting leaderboard...${NC}"
curl -s -X POST "$BASE_URL/api/v1/leaderboards/$LEADERBOARD_ID/reset" > /dev/null 2>&1

get_player_name() {
  local idx=$1
  local prefix_idx=$((idx % ${#PREFIXES[@]}))
  local suffix=$((idx / ${#PREFIXES[@]} + 1))
  echo "${PREFIXES[$prefix_idx]}${suffix}"
}

# Create 1000 players with scores between 5000-10000
echo -e "${CYAN}Creating $TOTAL_PLAYERS players...${NC}"
for batch_start in $(seq 0 100 999); do
  batch_end=$((batch_start + 99))
  scores="["
  first=true
  
  for i in $(seq $batch_start $batch_end); do
    player=$(get_player_name $i)
    score=$((RANDOM % 5000 + 5000))  # 5000-10000
    
    if [ "$first" = true ]; then first=false; else scores="$scores,"; fi
    scores="$scores{\"player_id\":\"$player\",\"leaderboard_id\":\"$LEADERBOARD_ID\",\"score\":$score}"
  done
  scores="$scores]"
  
  curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
    -H "Content-Type: application/json" \
    -d "{\"scores\":$scores}" > /dev/null
  
  echo -ne "\r  Progress: ${GREEN}$((batch_end + 1))${NC}/$TOTAL_PLAYERS"
done
echo -e "\n${GREEN}✓ Created $TOTAL_PLAYERS players${NC}\n"

# Show initial top 10
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
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}⚔️  Starting Battle! Scores changing constantly...${NC}"
echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Press Ctrl+C to stop"
echo ""

batch_num=0
while true; do
  scores="["
  first=true
  
  for i in $(seq 1 10); do
    # Mix of players: 50% from top 50, 50% from rest
    if [ $((RANDOM % 2)) -eq 0 ]; then
      player_idx=$((RANDOM % 50))
    else
      player_idx=$((RANDOM % 950 + 50))
    fi
    
    player=$(get_player_name $player_idx)
    
    # High variance scores: 6000-12000 (can overtake top players!)
    score=$((RANDOM % 6000 + 6000))
    
    if [ "$first" = true ]; then first=false; else scores="$scores,"; fi
    scores="$scores{\"player_id\":\"$player\",\"leaderboard_id\":\"$LEADERBOARD_ID\",\"score\":$score}"
  done
  scores="$scores]"
  
  curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
    -H "Content-Type: application/json" \
    -d "{\"scores\":$scores}" > /dev/null
  
  batch_num=$((batch_num + 1))
  
  if [ $((batch_num % 5)) -eq 0 ]; then
    timestamp=$(date +"%H:%M:%S")
    echo -e "[$timestamp] ⚔️  Batch #$batch_num | ${CYAN}$((batch_num * 10))${NC} score updates"
  fi
  
  sleep 1
done

