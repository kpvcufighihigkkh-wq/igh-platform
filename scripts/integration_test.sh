#!/bin/bash
# IGH Platform - Full Integration Test
# Simulates complete production flow: doffing → trolley → warehouse → sorting → weighing → packing → pallet

set -e
BASE="http://localhost:8099"
API_KEY="sk-igh-dev-key-change-in-production"
PASS=0
FAIL=0

check() {
  local desc="$1" expected="$2" actual="$3"
  if echo "$actual" | grep -q "$expected"; then
    echo "  [PASS] $desc"
    PASS=$((PASS+1))
  else
    echo "  [FAIL] $desc (expected: $expected, got: $actual)"
    FAIL=$((FAIL+1))
  fi
}

edge_post() {
  curl -s -X POST "$BASE/api/v1/edge/upload" \
    -H "X-API-Key: $API_KEY" -H "Content-Type: application/json" \
    -d "$1"
}

api_get() {
  curl -s "$BASE/api/v1$1" -H "Authorization: Bearer $TOKEN"
}

api_post() {
  curl -s -X POST "$BASE/api/v1$1" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2"
}

api_put() {
  curl -s -X PUT "$BASE/api/v1$1" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$2"
}

echo "========================================="
echo " IGH Platform Integration Test"
echo "========================================="
echo ""

# 0. Health & Login
echo "[Step 0] Health & Auth"
R=$(curl -s "$BASE/health")
check "health endpoint" '"status":"ok"' "$R"

R=$(curl -s -X POST "$BASE/api/v1/auth/login" -H "Content-Type: application/json" -d '{"username":"admin","password":"test"}')
TOKEN=$(echo "$R" | sed 's/.*"token":"\([^"]*\)".*/\1/')
check "login returns JWT" "eyJ" "$TOKEN"

# 1. Setup: Create lot + production line
echo ""
echo "[Step 1] Setup: Lot + Line"
R=$(api_post "/lots" '{"code":"TEST-LOT-001","specification":"40D/34F","product_type":"FDY","status":"active"}')
check "create lot" '"code":"TEST-LOT-001"' "$R"
LOT_ID=$(echo "$R" | sed 's/.*"id":\([0-9]*\).*/\1/')

# 2. Doffing event
echo ""
echo "[Step 2] Doffing (edge event)"
R=$(edge_post '{"edge_id":"edge-winding-01","event_type":"doffing","timestamp":"2026-05-20T08:00:00Z","payload":{"line_code":"L01","winder_name":"WDC-L01-1A","doff_no":1,"lot_code":"TEST-LOT-001","bobbin_count":8}}')
check "doffing event accepted" '"received":true' "$R"
sleep 1

R=$(api_get "/doffings?lot_id=$LOT_ID")
check "doffing recorded" '"total":1' "$R"

R=$(api_get "/bobbins?lot_id=$LOT_ID")
check "8 bobbins created" '"total":8' "$R"

# 3. Trolley loaded
echo ""
echo "[Step 3] Trolley Loaded (edge event)"
R=$(edge_post '{"edge_id":"edge-winding-01","event_type":"trolley_loaded","timestamp":"2026-05-20T08:10:00Z","payload":{"trolley_number":101,"rfid":"RFID-101","lot_code":"TEST-LOT-001","line_code":"L01","doffing_ids":[1]}}')
check "trolley event accepted" '"received":true' "$R"
sleep 1

R=$(api_get "/trolleys?lot_id=$LOT_ID")
check "trolley created" '"number":101' "$R"

# 4. Warehouse inbound
echo ""
echo "[Step 4] Warehouse Inbound"
R=$(edge_post '{"edge_id":"edge-warehouse-01","event_type":"warehouse_in","timestamp":"2026-05-20T08:15:00Z","payload":{"warehouse_code":"WH-TROLLEY-A","slot":{"row":2,"col":3,"layer":1},"occupant_type":"trolley","occupant_id":1,"lot_id":1}}')
check "warehouse_in accepted" '"received":true' "$R"
sleep 1

R=$(api_get "/warehouses/1/stats")
check "warehouse has occupied slot" '"occupied"' "$R"

# 5. Warehouse outbound
echo ""
echo "[Step 5] Warehouse Outbound"
R=$(edge_post '{"edge_id":"edge-warehouse-01","event_type":"warehouse_out","timestamp":"2026-05-20T09:00:00Z","payload":{"warehouse_code":"WH-TROLLEY-A","slot":{"row":2,"col":3,"layer":1},"occupant_id":1}}')
check "warehouse_out accepted" '"received":true' "$R"

# 6. Sorting
echo ""
echo "[Step 6] Sorting (quality)"
R=$(edge_post '{"edge_id":"edge-quality-01","event_type":"sorting_result","timestamp":"2026-05-20T09:10:00Z","payload":{"bobbin_id":1,"sorting_grade":"AA","defect_code":""}}')
check "sorting accepted" '"received":true' "$R"

# 7. Weighing
echo ""
echo "[Step 7] Weighing (quality)"
R=$(edge_post '{"edge_id":"edge-quality-01","event_type":"weighing_result","timestamp":"2026-05-20T09:11:00Z","payload":{"bobbin_id":1,"weight":3.456}}')
check "weighing accepted" '"received":true' "$R"

# 8. Packing complete
echo ""
echo "[Step 8] Packing Complete"
R=$(edge_post '{"edge_id":"edge-packaging-01","event_type":"packing_complete","timestamp":"2026-05-20T10:00:00Z","payload":{"pallet_code":"PLT-TEST-001","lot_code":"TEST-LOT-001","grade":"AA","bobbin_count":48,"layers":6,"total_weight":165.3}}')
check "packing_complete accepted" '"received":true' "$R"
sleep 1

R=$(api_get "/pallets")
check "pallet created" '"PLT-TEST-001"' "$R"

# 9. Alarm flow
echo ""
echo "[Step 9] Alarm Flow"
R=$(edge_post '{"edge_id":"edge-winding-01","event_type":"alarm","timestamp":"2026-05-20T10:30:00Z","payload":{"alarm_code":"TEST_ALARM","level":"normal","message":"Test alarm","device_code":"plc-L01"}}')
check "alarm event accepted" '"received":true' "$R"
sleep 1

R=$(api_get "/alarms/active-count")
check "active alarm count > 0" '"count"' "$R"

# 10. Metrics
echo ""
echo "[Step 10] Prometheus Metrics"
R=$(curl -s "$BASE/metrics" | grep "igh_http_requests_total" | head -1)
check "metrics exposed" "igh_http_requests_total" "$R"

# 11. Modules API
echo ""
echo "[Step 11] System Modules"
R=$(api_get "/system/modules")
check "modules config" '"Winding":true' "$R"

# Summary
echo ""
echo "========================================="
echo " Results: $PASS passed, $FAIL failed"
echo "========================================="
[ $FAIL -eq 0 ] && exit 0 || exit 1
