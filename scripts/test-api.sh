#!/bin/bash
# Leaderboard API Test Script
# Run: chmod +x scripts/test-api.sh && ./scripts/test-api.sh

# Don't exit on error - we want to track pass/fail

BASE_URL="${BASE_URL:-http://localhost:8080}"
PASSED=0
FAILED=0

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Leaderboard API Test Suite"
echo "Base URL: $BASE_URL"
echo "=========================================="

# Helper function to run tests
run_test() {
    local name="$1"
    local expected="$2"
    local response="$3"
    
    if echo "$response" | grep -q "$expected"; then
        echo -e "${GREEN}✅ PASS${NC}: $name"
        ((PASSED++))
    else
        echo -e "${RED}❌ FAIL${NC}: $name"
        echo "Expected to contain: $expected"
        echo "Got: $response"
        ((FAILED++))
    fi
}

# Test 1: Health Check
echo -e "\n${YELLOW}--- Test 1: Health Check ---${NC}"
response=$(curl -s "$BASE_URL/health")
run_test "Health Check" '"success":true' "$response"

# Test 2: Ready Check
echo -e "\n${YELLOW}--- Test 2: Ready Check ---${NC}"
response=$(curl -s "$BASE_URL/ready")
run_test "Ready Check" '"success":true' "$response"

# Cleanup: Delete test leaderboards if they exist
curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/test-game1" > /dev/null 2>&1 || true
curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/test-game2" > /dev/null 2>&1 || true

# Test 3: Create Leaderboard (Best Mode)
echo -e "\n${YELLOW}--- Test 3: Create Leaderboard (Best Mode) ---${NC}"
response=$(curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "test-game1", "name": "Test Game 1", "sort_order": "desc", "update_mode": "best"}')
run_test "Create Leaderboard" '"id":"test-game1"' "$response"

# Test 4: Submit Batch Scores
echo -e "\n${YELLOW}--- Test 4: Submit Batch Scores ---${NC}"
response=$(curl -s -X POST "$BASE_URL/api/v1/scores/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "scores": [
      {"player_id": "player1", "leaderboard_id": "test-game1", "score": 1000},
      {"player_id": "player2", "leaderboard_id": "test-game1", "score": 2500},
      {"player_id": "player3", "leaderboard_id": "test-game1", "score": 1800},
      {"player_id": "player4", "leaderboard_id": "test-game1", "score": 3200},
      {"player_id": "player5", "leaderboard_id": "test-game1", "score": 950}
    ]
  }')
run_test "Submit Batch Scores" '"received":5' "$response"

# Test 5: Get Top Players
echo -e "\n${YELLOW}--- Test 5: Get Top Players ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/top?limit=10")
run_test "Get Top Players" '"rank":1' "$response"
run_test "Top Player is player4" '"player_id":"player4"' "$response"

# Test 6: Get Player Rank
echo -e "\n${YELLOW}--- Test 6: Get Player Rank ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/player/player3")
run_test "Get Player Rank" '"rank":3' "$response"

# Test 7: Get Players Around
echo -e "\n${YELLOW}--- Test 7: Get Players Around ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/around/player3?range=2")
run_test "Get Players Around" '"success":true' "$response"

# Test 8: Get Leaderboard Stats
echo -e "\n${YELLOW}--- Test 8: Get Leaderboard Stats ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/stats")
run_test "Get Stats" '"total_players":5' "$response"

# Test 9: Best Mode - Lower Score Should NOT Update
echo -e "\n${YELLOW}--- Test 9: Best Mode - Lower Score ---${NC}"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player4", "leaderboard_id": "test-game1", "score": 2000}' > /dev/null
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/player/player4")
run_test "Lower score NOT updated" '"score":3200' "$response"

# Test 10: Best Mode - Higher Score Should Update
echo -e "\n${YELLOW}--- Test 10: Best Mode - Higher Score ---${NC}"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player4", "leaderboard_id": "test-game1", "score": 5000}' > /dev/null
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/player/player4")
run_test "Higher score updated" '"score":5000' "$response"

# Test 11: List Leaderboards
echo -e "\n${YELLOW}--- Test 11: List Leaderboards ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards")
run_test "List Leaderboards" '"test-game1"' "$response"

# Test 12: Remove Player
echo -e "\n${YELLOW}--- Test 12: Remove Player ---${NC}"
response=$(curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/test-game1/player/player5")
run_test "Remove Player" '"status":"removed"' "$response"

# Test 13: Get Rank Range
echo -e "\n${YELLOW}--- Test 13: Get Rank Range ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/range?start=1&end=3")
run_test "Get Rank Range" '"success":true' "$response"

# Test 14: Error - Non-existent Player
echo -e "\n${YELLOW}--- Test 14: Error - Non-existent Player ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game1/player/nonexistent")
run_test "Non-existent Player Error" '"success":false' "$response"

# Test 15: Error - Non-existent Leaderboard
echo -e "\n${YELLOW}--- Test 15: Error - Non-existent Leaderboard ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/nonexistent")
run_test "Non-existent Leaderboard Error" '"success":false' "$response"

# Test 16: Create Increment Mode Leaderboard
echo -e "\n${YELLOW}--- Test 16: Create Increment Mode Leaderboard ---${NC}"
response=$(curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "test-game2", "name": "Test Game 2 - Increment", "update_mode": "increment"}')
run_test "Create Increment Leaderboard" '"update_mode":"increment"' "$response"

# Test 17: Increment Mode
echo -e "\n${YELLOW}--- Test 17: Increment Mode ---${NC}"
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player1", "leaderboard_id": "test-game2", "score": 100}' > /dev/null
curl -s -X POST "$BASE_URL/api/v1/scores" \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player1", "leaderboard_id": "test-game2", "score": 50}' > /dev/null
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game2/player/player1")
run_test "Increment Mode (100+50=150)" '"score":150' "$response"

# Test 18: Duplicate Leaderboard Error
echo -e "\n${YELLOW}--- Test 18: Duplicate Leaderboard Error ---${NC}"
response=$(curl -s -X POST "$BASE_URL/api/v1/leaderboards" \
  -H "Content-Type: application/json" \
  -d '{"id": "test-game1", "name": "Duplicate"}')
run_test "Duplicate Leaderboard Error" '"leaderboard already exists"' "$response"

# Test 19: Reset Leaderboard
echo -e "\n${YELLOW}--- Test 19: Reset Leaderboard ---${NC}"
response=$(curl -s -X POST "$BASE_URL/api/v1/leaderboards/test-game2/reset")
run_test "Reset Leaderboard" '"status":"reset"' "$response"

# Test 20: Verify Reset (Empty)
echo -e "\n${YELLOW}--- Test 20: Verify Reset ---${NC}"
response=$(curl -s "$BASE_URL/api/v1/leaderboards/test-game2/top?limit=10")
run_test "Leaderboard Empty After Reset" '"data":\[\]' "$response"

# Cleanup
echo -e "\n${YELLOW}--- Cleanup ---${NC}"
curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/test-game1" > /dev/null 2>&1 || true
curl -s -X DELETE "$BASE_URL/api/v1/leaderboards/test-game2" > /dev/null 2>&1 || true
echo "Test leaderboards cleaned up"

# Summary
echo ""
echo "=========================================="
echo "TEST RESULTS"
echo "=========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo "Total: $((PASSED + FAILED))"
echo "=========================================="

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi

