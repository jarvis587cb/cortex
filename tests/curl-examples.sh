#!/bin/bash

# Cortex API Test-Script mit curl
# Beispiele für alle API-Endpunkte

API_URL="${CORTEX_API_URL:-http://localhost:9123}"
APP_ID="${CORTEX_APP_ID:-openclaw}"
USER_ID="${CORTEX_USER_ID:-default}"

# Farben
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Cortex API Tests ===${NC}\n"

# Health Check
echo -e "${BLUE}1. Health Check${NC}"
curl -s "$API_URL/health" | jq '.' || curl -s "$API_URL/health"
echo -e "\n"

# Memory speichern (Seeds-API)
echo -e "${BLUE}2. Memory speichern (Seeds-API)${NC}"
STORE_RESPONSE=$(curl -s -X POST "$API_URL/seeds" \
    -H "Content-Type: application/json" \
    -d "{
        \"appId\": \"$APP_ID\",
        \"externalUserId\": \"$USER_ID\",
        \"content\": \"Test Memory: $(date)\",
        \"metadata\": {\"tags\": [\"test\", \"curl\"]}
    }")
echo "$STORE_RESPONSE" | jq '.' || echo "$STORE_RESPONSE"
MEMORY_ID=$(echo "$STORE_RESPONSE" | jq -r '.id // empty' 2>/dev/null || echo "")
echo -e "\n"

# Memory-Suche
echo -e "${BLUE}3. Memory-Suche${NC}"
curl -s -X POST "$API_URL/seeds/query" \
    -H "Content-Type: application/json" \
    -d "{
        \"appId\": \"$APP_ID\",
        \"externalUserId\": \"$USER_ID\",
        \"query\": \"Test\",
        \"limit\": 5
    }" | jq '.' || curl -s -X POST "$API_URL/seeds/query" \
    -H "Content-Type: application/json" \
    -d "{
        \"appId\": \"$APP_ID\",
        \"externalUserId\": \"$USER_ID\",
        \"query\": \"Test\",
        \"limit\": 5
    }"
echo -e "\n"

# Memory löschen (wenn ID vorhanden)
if [ -n "$MEMORY_ID" ] && [ "$MEMORY_ID" != "null" ] && [ "$MEMORY_ID" != "" ]; then
    echo -e "${BLUE}4. Memory löschen (ID: $MEMORY_ID)${NC}"
    curl -s -X DELETE "$API_URL/seeds/$MEMORY_ID?appId=$APP_ID&externalUserId=$USER_ID" | jq '.' || curl -s -X DELETE "$API_URL/seeds/$MEMORY_ID?appId=$APP_ID&externalUserId=$USER_ID"
    echo -e "\n"
fi

# Cortex API - Remember
echo -e "${BLUE}5. Erinnerung speichern (Cortex API)${NC}"
curl -s -X POST "$API_URL/remember" \
    -H "Content-Type: application/json" \
    -d "{
        \"content\": \"Cortex API Test: $(date)\",
        \"type\": \"semantic\",
        \"tags\": \"test,curl\",
        \"importance\": 5
    }" | jq '.' || curl -s -X POST "$API_URL/remember" \
    -H "Content-Type: application/json" \
    -d "{
        \"content\": \"Cortex API Test: $(date)\",
        \"type\": \"semantic\",
        \"tags\": \"test,curl\",
        \"importance\": 5
    }"
echo -e "\n"

# Recall
echo -e "${BLUE}6. Erinnerungen abrufen${NC}"
curl -s "$API_URL/recall?q=Test&limit=5" | jq '.' || curl -s "$API_URL/recall?q=Test&limit=5"
echo -e "\n"

# Entity - Fakt setzen
echo -e "${BLUE}7. Fakt für Entity setzen${NC}"
curl -s -X POST "$API_URL/entities?entity=user:test" \
    -H "Content-Type: application/json" \
    -d "{
        \"key\": \"test_key\",
        \"value\": \"test_value\"
    }"
echo -e "\n"

# Entity abrufen
echo -e "${BLUE}8. Entity abrufen${NC}"
curl -s "$API_URL/entities?name=user:test" | jq '.' || curl -s "$API_URL/entities?name=user:test"
echo -e "\n"

# Relation hinzufügen
echo -e "${BLUE}9. Relation hinzufügen${NC}"
curl -s -X POST "$API_URL/relations" \
    -H "Content-Type: application/json" \
    -d "{
        \"from\": \"user:test1\",
        \"to\": \"user:test2\",
        \"type\": \"test\"
    }"
echo -e "\n"

# Statistiken
echo -e "${BLUE}10. Statistiken${NC}"
curl -s "$API_URL/stats" | jq '.' || curl -s "$API_URL/stats"
echo -e "\n"

echo -e "${GREEN}=== Tests abgeschlossen ===${NC}"
