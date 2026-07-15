#!/usr/bin/env bash
# Create a ChatGPT Subscription (Codex) channel and run feat-codex-001 checks (C1–C4).
#
# Usage:
#   export RELAY_URL={BASE_URL}
#   export ADMIN_USER=root
#   export ADMIN_PASSWORD='your-admin-password'
#   export CODEX_OAUTH_FILE=/path/to/codex-oauth.json
#   export CHANNEL_MODELS=gpt-5.3-codex   # optional
#   export CHANNEL_NAME='Plus Codex 1'      # optional
#   ./scripts/setup-codex-channel.sh
#
# Requires: curl, jq

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PUBLIC_URL="${RELAY_URL:-}"
ADMIN_USER="${ADMIN_USER:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
CODEX_OAUTH_FILE="${CODEX_OAUTH_FILE:-}"
CHANNEL_NAME="${CHANNEL_NAME:-Plus Codex 1}"
CHANNEL_MODELS="${CHANNEL_MODELS:-gpt-5.3-codex}"
CHANNEL_GROUP="${CHANNEL_GROUP:-default}"
CHANNEL_WEIGHT="${CHANNEL_WEIGHT:-1}"
TIMEOUT="${RELAY_TIMEOUT:-120}"
COOKIE_JAR=""
TMP_DIR=""
FAIL=0
CHANNEL_ID=""

cleanup() {
  if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage: setup-codex-channel.sh

Environment:
  RELAY_URL           Relay base URL (required)
  ADMIN_USER          Admin username (required)
  ADMIN_PASSWORD      Admin password (required)
  CODEX_OAUTH_FILE    Path to Plus OAuth JSON (required)
  CHANNEL_MODELS      Comma-separated model ids (default: gpt-5.3-codex)
  CHANNEL_NAME        Channel display name (default: Plus Codex 1)
  CHANNEL_GROUP       Channel group (default: default)
  CHANNEL_WEIGHT      Routing weight (default: 1)

OAuth JSON must include: access_token, account_id, refresh_token
EOF
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

check() {
  local name="$1"
  shift
  if "$@"; then
    echo "[PASS] $name"
  else
    echo "[FAIL] $name" >&2
    FAIL=1
  fi
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

require_cmd curl
require_cmd jq

if [[ -z "$PUBLIC_URL" || -z "$ADMIN_USER" || -z "$ADMIN_PASSWORD" || -z "$CODEX_OAUTH_FILE" ]]; then
  usage
  die "RELAY_URL, ADMIN_USER, ADMIN_PASSWORD, and CODEX_OAUTH_FILE are required"
fi

[[ -f "$CODEX_OAUTH_FILE" ]] || die "OAuth file not found: $CODEX_OAUTH_FILE"

PUBLIC_URL="${PUBLIC_URL%/}"
TMP_DIR="$(mktemp -d)"
COOKIE_JAR="$TMP_DIR/cookies.txt"

echo "=== feat-codex-001: Codex Plus channel setup ==="
echo "URL:   $PUBLIC_URL"
echo "OAuth: $CODEX_OAUTH_FILE"
echo "Model: $CHANNEL_MODELS"
echo ""

# --- C4: validate OAuth JSON fields ---
echo "=== C4: OAuth JSON validation ==="
OAUTH_KEY="$(jq -c '.' "$CODEX_OAUTH_FILE")"
check "access_token present" jq -e '.access_token | type == "string" and length > 0' "$CODEX_OAUTH_FILE"
check "account_id present" jq -e '.account_id | type == "string" and length > 0' "$CODEX_OAUTH_FILE"
check "refresh_token present" jq -e '.refresh_token | type == "string" and length > 0' "$CODEX_OAUTH_FILE"

# --- Admin login ---
echo ""
echo "=== Admin login ==="
LOGIN_FILE="$TMP_DIR/login.json"
LOGIN_CODE="$(curl -sS --max-time "$TIMEOUT" -c "$COOKIE_JAR" \
  -o "$LOGIN_FILE" -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
  "${PUBLIC_URL}/api/user/login")"

check "POST /api/user/login HTTP 200 (got $LOGIN_CODE)" test "$LOGIN_CODE" = "200"
check "login success:true" jq -e '.success == true' "$LOGIN_FILE"

if jq -e '.data.require_2fa == true' "$LOGIN_FILE" >/dev/null 2>&1; then
  die "Admin account requires 2FA; complete login in browser or disable 2FA for automation"
fi

# --- Create channel ---
echo ""
echo "=== Create Codex channel (type=57) ==="
CREATE_BODY="$TMP_DIR/create.json"
CREATE_RESP="$TMP_DIR/create_resp.json"
jq -n \
  --arg key "$OAUTH_KEY" \
  --arg name "$CHANNEL_NAME" \
  --arg models "$CHANNEL_MODELS" \
  --arg group "$CHANNEL_GROUP" \
  --argjson weight "$CHANNEL_WEIGHT" \
  '{
    mode: "single",
    channel: {
      type: 57,
      key: $key,
      name: $name,
      models: $models,
      group: $group,
      weight: $weight,
      status: 1
    }
  }' > "$CREATE_BODY"

CREATE_CODE="$(curl -sS --max-time "$TIMEOUT" -b "$COOKIE_JAR" \
  -o "$CREATE_RESP" -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  -d @"$CREATE_BODY" \
  "${PUBLIC_URL}/api/channel/")"

check "POST /api/channel/ HTTP 200 (got $CREATE_CODE)" test "$CREATE_CODE" = "200"
check "channel create success:true" jq -e '.success == true' "$CREATE_RESP"

CHANNEL_ID="$(jq -r '.data.id // .data[0].id // empty' "$CREATE_RESP" 2>/dev/null || true)"
if [[ -z "$CHANNEL_ID" ]]; then
  # fallback: list channels and find by name
  LIST_FILE="$TMP_DIR/channels.json"
  curl -sS --max-time "$TIMEOUT" -b "$COOKIE_JAR" \
    "${PUBLIC_URL}/api/channel/?type=57&page_size=100" -o "$LIST_FILE"
  CHANNEL_ID="$(jq -r --arg n "$CHANNEL_NAME" '.data.items[]? | select(.name == $n) | .id' "$LIST_FILE" | head -1)"
fi

[[ -n "$CHANNEL_ID" ]] || die "Could not determine channel id after create"
echo "Channel id: $CHANNEL_ID"

# --- C1: channel test ---
echo ""
echo "=== C1: channel test (/v1/responses stream) ==="
TEST_FILE="$TMP_DIR/test.json"
TEST_CODE="$(curl -sS --max-time "$TIMEOUT" -b "$COOKIE_JAR" \
  -o "$TEST_FILE" -w '%{http_code}' \
  "${PUBLIC_URL}/api/channel/test/${CHANNEL_ID}")"
check "GET /api/channel/test/${CHANNEL_ID} HTTP 200 (got $TEST_CODE)" test "$TEST_CODE" = "200"
check "channel test success:true" jq -e '.success == true' "$TEST_FILE"

# --- C2: refresh credential ---
echo ""
echo "=== C2: refresh credential ==="
REFRESH_FILE="$TMP_DIR/refresh.json"
REFRESH_CODE="$(curl -sS --max-time "$TIMEOUT" -b "$COOKIE_JAR" \
  -o "$REFRESH_FILE" -w '%{http_code}' \
  -X POST "${PUBLIC_URL}/api/channel/${CHANNEL_ID}/codex/refresh")"
check "POST /api/channel/${CHANNEL_ID}/codex/refresh HTTP 200 (got $REFRESH_CODE)" test "$REFRESH_CODE" = "200"
check "refresh success:true" jq -e '.success == true' "$REFRESH_FILE"
check "expires_at updated" jq -e '.data.expires_at != null and (.data.expires_at | tostring | length > 0)' "$REFRESH_FILE"
EXPIRES_AT="$(jq -r '.data.expires_at // empty' "$REFRESH_FILE")"
[[ -n "$EXPIRES_AT" ]] && echo "expires_at: $EXPIRES_AT"

# --- C3: usage API ---
echo ""
echo "=== C3: Codex usage ==="
USAGE_FILE="$TMP_DIR/usage.json"
USAGE_CODE="$(curl -sS --max-time "$TIMEOUT" -b "$COOKIE_JAR" \
  -o "$USAGE_FILE" -w '%{http_code}' \
  "${PUBLIC_URL}/api/channel/${CHANNEL_ID}/codex/usage")"
check "GET /api/channel/${CHANNEL_ID}/codex/usage HTTP 200 (got $USAGE_CODE)" test "$USAGE_CODE" = "200"
check "usage success:true" jq -e '.success == true' "$USAGE_FILE"

echo ""
if [[ "$FAIL" -eq 0 ]]; then
  echo "=== feat-codex-001 COMPLETE (C1–C4 PASS) ==="
  echo "Channel id: $CHANNEL_ID"
  echo "Next: feat-codex-002 (system policies) — see docs/installation/codex-relay-production.md §3.3"
  exit 0
fi

echo "=== Some checks failed ==="
echo "See docs/installation/codex-relay-production.md for remediation."
exit 1
