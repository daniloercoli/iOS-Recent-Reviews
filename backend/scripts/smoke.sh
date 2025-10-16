#!/usr/bin/env bash
set -euo pipefail

# Default Configurazion
BASE_URL="${BASE_URL:-http://localhost:8080}"
APP_ID="${APP_ID:-595068606}"
COUNTRY="${COUNTRY:-us}"
HOURS="${HOURS:-48}"

echo "== Smoke test backend =="
echo "BASE_URL=$BASE_URL  APP_ID=$APP_ID  COUNTRY=$COUNTRY  HOURS=$HOURS"
echo

# Helper for assert
assert_eq() {
  local got="$1" exp="$2" msg="$3"
  if [[ "$got" != "$exp" ]]; then
    echo "❌ $msg (got: '$got', expected: '$exp')" >&2
    exit 1
  fi
  echo "✅ $msg"
}

assert_nonempty() {
  local got="$1" msg="$2"
  if [[ -z "$got" ]]; then
    echo "❌ $msg (got empty)" >&2
    exit 1
  fi
  echo "✅ $msg"
}

# 1) /health
echo "--> GET /health"
HEALTH_JSON=$(http --check-status --pretty=none --print=b GET "$BASE_URL/health")
HEALTH_STATUS=$(echo "$HEALTH_JSON" | jq -r '.status')
assert_eq "$HEALTH_STATUS" "ok" "/health status ok"
echo

# 2) /apps
echo "--> GET /apps"
APPS_JSON=$(http --check-status --pretty=none --print=b GET "$BASE_URL/apps")
APPS_COUNT=$(echo "$APPS_JSON" | jq 'length')
echo "apps configured: $APPS_COUNT"
# Don't fail if 0, but verify it's valid JSON
assert_nonempty "$APPS_JSON" "/apps returns JSON"
echo

# 3) /poll (async) – optional; don't fail if not configured
echo "--> POST /poll?appId=$APP_ID&country=$COUNTRY"
http --ignore-stdin --check-status --print=h POST "$BASE_URL/poll" appId=="$APP_ID" country=="$COUNTRY" \
  || echo "WARN: /poll may be disabled or only GET. Trying GET…"
http --ignore-stdin --check-status --print=h GET "$BASE_URL/poll?appId=$APP_ID&country=$COUNTRY" \
  || echo "WARN: /poll GET failed (ok if not supported)"
echo

# 4)/reviews?hours=… (with 2-3 light retries, since /poll is async)
echo "--> GET /reviews (with short retries)"
ATTEMPTS=4
DELAY=1
REVIEWS_JSON=""
for ((i=1; i<=ATTEMPTS; i++)); do
  set +e
  REVIEWS_JSON=$(http --check-status --pretty=none --print=b GET \
    "$BASE_URL/reviews" appId=="$APP_ID" country=="$COUNTRY" hours=="$HOURS" 2>/dev/null)
  CODE=$?
  set -e
  if [[ $CODE -eq 0 ]]; then break; fi
  echo "retry $i/$ATTEMPTS after ${DELAY}s…"
  sleep "$DELAY"
  DELAY=$((DELAY*2))
done
assert_nonempty "$REVIEWS_JSON" "/reviews returned JSON"

COUNT=$(echo "$REVIEWS_JSON" | jq -r '.count')
FROM=$(echo "$REVIEWS_JSON" | jq -r '.from')
TO=$(echo "$REVIEWS_JSON" | jq -r '.to')
echo "window: $FROM → $TO ; count=$COUNT"

# Basic field validation on first element (if present)
if [[ "$COUNT" -gt 0 ]]; then
  FIRST_ID=$(echo "$REVIEWS_JSON" | jq -r '.reviews[0].id')
  FIRST_TS=$(echo "$REVIEWS_JSON" | jq -r '.reviews[0].submittedAt')
  FIRST_RATING=$(echo "$REVIEWS_JSON" | jq -r '.reviews[0].rating')
  assert_nonempty "$FIRST_ID" "first review has id"
  assert_nonempty "$FIRST_TS" "first review has submittedAt"
  assert_nonempty "$FIRST_RATING" "first review has rating"
fi

echo
echo "== Done. All good! =="
