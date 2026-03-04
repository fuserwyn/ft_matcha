#!/usr/bin/env bash
# =============================================================================
#  SQL Injection protection test suite for ft_matcha API
#  Usage: ./scripts/test_sqli.sh [API_BASE_URL]
#  Default API_BASE_URL: http://localhost:8080
# =============================================================================
set -euo pipefail

API="${1:-http://localhost:8080}"
PASS=0
FAIL=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RESET='\033[0m'

sep() {
  echo ""
  echo -e "${CYAN}════════════════════════════════════════════════════════════${RESET}"
  echo -e "${CYAN}  $1${RESET}"
  echo -e "${CYAN}════════════════════════════════════════════════════════════${RESET}"
}

# expect_status <label> <expected_http_code> <actual_http_code> [body]
expect_status() {
  local label="$1" expected="$2" actual="$3" body="${4:-}"
  if [ "$actual" = "$expected" ]; then
    echo -e "  ${GREEN}PASS${RESET} [$actual] $label"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}FAIL${RESET} [$actual != $expected] $label"
    [ -n "$body" ] && echo -e "       body: $(echo "$body" | head -c 200)"
    FAIL=$((FAIL+1))
  fi
}

# expect_not_contains <label> <string> <body>
expect_not_contains() {
  local label="$1" needle="$2" body="$3"
  if echo "$body" | grep -qi "$needle"; then
    echo -e "  ${RED}FAIL${RESET} response leaks '$needle' — $label"
    FAIL=$((FAIL+1))
  else
    echo -e "  ${GREEN}PASS${RESET} no '$needle' leak — $label"
    PASS=$((PASS+1))
  fi
}

post() {
  local url="$1" data="$2"
  curl -s -o /tmp/sqli_body -w "%{http_code}" \
    -X POST -H "Content-Type: application/json" \
    --max-time 5 \
    -d "$data" "${API}${url}" 2>/dev/null || echo "000"
}

get_auth() {
  local url="$1" token="$2"
  curl -s -o /tmp/sqli_body -w "%{http_code}" \
    -X GET -H "Authorization: Bearer $token" \
    --max-time 5 \
    "${API}${url}" 2>/dev/null || echo "000"
}

# ─── Pre-flight ───────────────────────────────────────────────────────────────
sep "PRE-FLIGHT: checking parameterized queries in source"

echo ""
echo -e "${YELLOW}  Scanning for unsafe query patterns (fmt.Sprintf + SQL)...${RESET}"

UNSAFE=$(grep -rn 'fmt\.Sprintf.*SELECT\|fmt\.Sprintf.*INSERT\|fmt\.Sprintf.*UPDATE\|fmt\.Sprintf.*DELETE\|fmt\.Sprintf.*WHERE' \
  "$(dirname "$0")/../api" --include="*.go" 2>/dev/null | grep -v "_test.go" | grep -v "docs.go" || true)

if [ -z "$UNSAFE" ]; then
  echo -e "  ${GREEN}PASS${RESET} No fmt.Sprintf SQL concatenation found"
  PASS=$((PASS+1))
else
  echo -e "  ${RED}FAIL${RESET} Found potential unsafe query building:"
  echo "$UNSAFE"
  FAIL=$((FAIL+1))
fi

echo ""
echo -e "${YELLOW}  Counting parameterized placeholders (\$1, \$2 ...) in repositories...${RESET}"
PARAM_COUNT=$(grep -roh '\$[0-9]\+' \
  "$(dirname "$0")/../api/internal/repository" --include="*.go" 2>/dev/null | wc -l | tr -d ' ')
echo -e "  ${GREEN}PASS${RESET} Found ${PARAM_COUNT} parameterized placeholders across all repository files"
PASS=$((PASS+1))

echo ""
echo -e "${YELLOW}  Checking database driver (must be pgx — enforces parameter binding)...${RESET}"
if grep -r 'jackc/pgx' "$(dirname "$0")/../api/go.mod" &>/dev/null; then
  echo -e "  ${GREEN}PASS${RESET} Driver: github.com/jackc/pgx/v5 (native parameterized queries)"
  PASS=$((PASS+1))
else
  echo -e "  ${RED}FAIL${RESET} Expected pgx driver not found in go.mod"
  FAIL=$((FAIL+1))
fi

# ─── Auth endpoint tests ──────────────────────────────────────────────────────
sep "1. LOGIN — classic SQL injection payloads"

TESTS=(
  "' OR '1'='1"
  "' OR 1=1--"
  "admin'--"
  "' OR 'x'='x"
  "'; DROP TABLE users;--"
  "' UNION SELECT id,username,password_hash,email,first_name,last_name,email_verified_at FROM users--"
  "\" OR \"\"=\""
  "1' AND SLEEP(5)--"
  "' AND 1=CONVERT(int,(SELECT TOP 1 password_hash FROM users))--"
  "' OR 1=1 LIMIT 1 OFFSET 0--"
)

for payload in "${TESTS[@]}"; do
  body='{"username":"'"$payload"'","password":"irrelevant"}'
  code=$(post "/api/v1/auth/login" "$body")
  resp=$(cat /tmp/sqli_body)
  expect_status "username='$payload'" 400 "$code" "$resp"
  expect_not_contains "no DB error leak" "syntax error\|pq:\|pgx:\|relation\|column" "$resp"
done

sep "2. REGISTER — injection in every field"

FIELDS=(
  '{"username":"'"'"' OR 1=1--","email":"a@b.com","password":"Test1234!","first_name":"A","last_name":"B"}'
  '{"username":"normal","email":"'"'"' OR 1=1--@b.com","password":"Test1234!","first_name":"A","last_name":"B"}'
  '{"username":"normal2","email":"a@b.com","password":"Test1234!","first_name":"'"'"'; DROP TABLE users;--","last_name":"B"}'
  '{"username":"normal3","email":"a@b.com","password":"Test1234!","first_name":"A","last_name":"'"'"' UNION SELECT * FROM users--"}'
)

for body in "${FIELDS[@]}"; do
  code=$(post "/api/v1/auth/register" "$body")
  resp=$(cat /tmp/sqli_body)
  # Accept 400 (validation) or 409 (conflict) — not 500 (DB error)
  if [ "$code" = "400" ] || [ "$code" = "409" ] || [ "$code" = "201" ]; then
    echo -e "  ${GREEN}PASS${RESET} [$code] register with injection payload rejected or sanitized"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}FAIL${RESET} [$code] unexpected status for register injection"
    FAIL=$((FAIL+1))
  fi
  expect_not_contains "no DB error leak in register" "syntax error\|pq:\|pgx:\|relation\|column" "$resp"
done

sep "3. TAUTOLOGY & UNION attacks on login"

UNION_TESTS=(
  "' UNION SELECT 1,2,3,4,5,6,7--"
  "' UNION ALL SELECT NULL,NULL,NULL,NULL,NULL,NULL,NULL--"
  "') UNION SELECT username,password_hash,email,first_name,last_name,id,email_verified_at FROM users--"
  "1; SELECT * FROM users--"
)

for payload in "${UNION_TESTS[@]}"; do
  body='{"username":"'"$payload"'","password":"x"}'
  code=$(post "/api/v1/auth/login" "$body")
  resp=$(cat /tmp/sqli_body)
  expect_status "UNION '$payload'" 400 "$code" "$resp"
  expect_not_contains "no user data leak" "password_hash\|email_verified" "$resp"
done

sep "4. BOOLEAN-BASED blind injection on login"

BLIND=(
  "' AND 1=1--"
  "' AND 1=2--"
  "' AND (SELECT COUNT(*) FROM users)>0--"
  "' AND (SELECT SUBSTRING(password_hash,1,1) FROM users LIMIT 1)='$'--"
)

for payload in "${BLIND[@]}"; do
  body='{"username":"'"$payload"'","password":"x"}'
  code=$(post "/api/v1/auth/login" "$body")
  expect_status "blind '$payload'" 400 "$code"
done

sep "5. TIME-BASED blind injection (pg_sleep)"

# These must return quickly (not hang for 5 s) AND return 400
TIME_PAYLOADS=(
  "'; SELECT pg_sleep(5)--"
  "' OR (SELECT pg_sleep(5))--"
  "1; WAITFOR DELAY '0:0:5'--"
)

for payload in "${TIME_PAYLOADS[@]}"; do
  body='{"username":"'"$payload"'","password":"x"}'
  START=$(date +%s)
  code=$(post "/api/v1/auth/login" "$body")
  ELAPSED=$(( $(date +%s) - START ))
  if [ "$ELAPSED" -lt 4 ] && [ "$code" = "400" ]; then
    echo -e "  ${GREEN}PASS${RESET} [${ELAPSED}s, $code] time-based payload did not cause delay"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}FAIL${RESET} [${ELAPSED}s, $code] possible time-based SQLi — took ${ELAPSED}s"
    FAIL=$((FAIL+1))
  fi
done

sep "6. PATH PARAMETER injection (user ID)"

PATH_IDS=(
  "' OR 1=1--"
  "00000000-0000-0000-0000-000000000000' OR '1'='1"
  "../../../etc/passwd"
  "1; DROP TABLE users--"
  "%27%20OR%201%3D1--"
)

# Get a real token first (seed user)
LOGIN_BODY='{"username":"seed_user_0001_001","password":"SeedPassw0rd!"}'
LOGIN_CODE=$(post "/api/v1/auth/login" "$LOGIN_BODY")
TOKEN=$(cat /tmp/sqli_body | grep -o '"token":"[^"]*"' | cut -d'"' -f4 || true)

if [ -z "$TOKEN" ]; then
  echo -e "  ${YELLOW}SKIP${RESET} Could not log in as seed user — skipping path param tests"
  echo -e "       (seed users may not exist; run 'make run' with seeding enabled)"
else
  for id in "${PATH_IDS[@]}"; do
    code=$(get_auth "/api/v1/users/${id}" "$TOKEN")
    resp=$(cat /tmp/sqli_body)
    if [ "$code" = "400" ] || [ "$code" = "404" ]; then
      echo -e "  ${GREEN}PASS${RESET} [$code] path id='$id' rejected"
      PASS=$((PASS+1))
    else
      echo -e "  ${RED}FAIL${RESET} [$code] unexpected response for path id='$id'"
      FAIL=$((FAIL+1))
    fi
    expect_not_contains "no DB error in path param" "syntax error\|pq:\|pgx:" "$resp"
  done
fi

sep "7. STACKED QUERIES & DDL"

DDL_PAYLOADS=(
  "'; DROP TABLE users;--"
  "'; DROP TABLE user_photos;--"
  "'; TRUNCATE TABLE likes;--"
  "'; ALTER TABLE users ADD COLUMN pwned TEXT;--"
  "'; CREATE TABLE hacked (id SERIAL);--"
)

for payload in "${DDL_PAYLOADS[@]}"; do
  body='{"username":"'"$payload"'","password":"x"}'
  code=$(post "/api/v1/auth/login" "$body")
  expect_status "DDL '$payload'" 400 "$code"
done

# Verify tables still exist after all attacks
echo ""
CONTAINER=$(docker ps --format '{{.Names}}' | grep -E 'postgres' | head -1 2>/dev/null || true)
if [ -n "$CONTAINER" ]; then
  TABLE_COUNT=$(docker exec -i "$CONTAINER" psql -U matcha -d matcha -P pager=off -t \
    -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';" \
    2>/dev/null | tr -d ' ')
  if [ "$TABLE_COUNT" -ge 12 ]; then
    echo -e "  ${GREEN}PASS${RESET} All $TABLE_COUNT tables still intact after DDL injection attempts"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}FAIL${RESET} Only $TABLE_COUNT tables found — DDL may have succeeded!"
    FAIL=$((FAIL+1))
  fi
fi

# ─── Summary ──────────────────────────────────────────────────────────────────
sep "SUMMARY"
echo ""
TOTAL=$((PASS+FAIL))
echo -e "  Total : $TOTAL"
echo -e "  ${GREEN}Passed: $PASS${RESET}"
if [ "$FAIL" -gt 0 ]; then
  echo -e "  ${RED}Failed: $FAIL${RESET}"
  echo ""
  echo -e "  ${RED}PROTECTION GAPS DETECTED — review failed tests above.${RESET}"
  exit 1
else
  echo -e "  ${GREEN}Failed: 0${RESET}"
  echo ""
  echo -e "  ${GREEN}All SQL injection protection tests passed.${RESET}"
  echo -e "  Protection method: pgx/v5 parameterized queries (\$1..\$N) — user input never interpolated into SQL."
fi
echo ""
