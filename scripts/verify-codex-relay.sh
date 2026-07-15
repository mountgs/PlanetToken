#!/usr/bin/env bash
# Codex CLI relay production verification.
#
# Validates infrastructure, token auth, model visibility, and /v1/responses
# (non-stream + stream) against a deployed new-api relay.
#
# Usage:
#   ./scripts/verify-codex-relay.sh --url {BASE_URL} --token sk-xxx --model gpt-5.3-codex
#   ./scripts/verify-codex-relay.sh --url http://127.0.0.1:3000 --phase infra
#   ./scripts/verify-codex-relay.sh --url https://api.example.com --token sk-xxx --model gpt-5.3-codex --phase all
#
# Phases:
#   infra  - /api/status, /v1/models auth gate (no token required for status)
#   auth   - valid/invalid token behavior on /v1/models
#   relay  - model list contains target codex model
#   codex  - POST /v1/responses (sync + SSE); requires --token and --model
#   all    - run every phase (default)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PUBLIC_URL="${RELAY_URL:-}"
USER_TOKEN="${RELAY_TOKEN:-}"
MODEL="${RELAY_MODEL:-gpt-5.3-codex}"
PHASE="all"
TIMEOUT="${RELAY_TIMEOUT:-120}"
FAIL=0
TMP_DIR=""

cleanup() {
  if [[ -n "$TMP_DIR" && -d "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage: verify-codex-relay.sh [options]

Options:
  --url URL          Relay base URL (no trailing slash). Env: RELAY_URL
  --token TOKEN      User API token (sk-...). Env: RELAY_TOKEN
  --model MODEL      Codex model id for E2E checks. Env: RELAY_MODEL (default: gpt-5.3-codex)
  --phase PHASE      infra | auth | relay | codex | all (default: all)
  --timeout SEC      curl max time per request (default: 120)
  -h, --help         Show this help

Examples:
  ./scripts/verify-codex-relay.sh --url http://127.0.0.1:3000 --phase infra
  ./scripts/verify-codex-relay.sh --url {BASE_URL} --token sk-xxx --model gpt-5.3-codex
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url) PUBLIC_URL="$2"; shift 2 ;;
    --token) USER_TOKEN="$2"; shift 2 ;;
    --model) MODEL="$2"; shift 2 ;;
    --phase) PHASE="$2"; shift 2 ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 1 ;;
  esac
done

if [[ -z "$PUBLIC_URL" ]]; then
  echo "ERROR: --url or RELAY_URL is required" >&2
  usage
  exit 1
fi

PUBLIC_URL="${PUBLIC_URL%/}"
TMP_DIR="$(mktemp -d)"

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

json_has_success_true() {
  local file="$1"
  grep -q '"success"[[:space:]]*:[[:space:]]*true' "$file"
}

json_has_field_nonempty() {
  local file="$1"
  local field="$2"
  if command -v jq >/dev/null 2>&1; then
    local val
    val="$(jq -r "$field // empty" "$file" 2>/dev/null || true)"
    [[ -n "$val" && "$val" != "null" ]]
  else
    grep -q "\"$(echo "$field" | sed 's/\.//g')\"" "$file" 2>/dev/null
  fi
}

curl_save() {
  local out="$1"
  shift
  curl -sS --max-time "$TIMEOUT" "$@" -o "$out" -w '%{http_code}'
}

phase_enabled() {
  local want="$1"
  [[ "$PHASE" == "all" || "$PHASE" == "$want" ]]
}

run_infra() {
  echo ""
  echo "=== Phase: infra ==="

  local status_file="$TMP_DIR/status.json"
  local code
  code="$(curl_save "$status_file" "${PUBLIC_URL}/api/status")"
  check "GET /api/status returns HTTP 200 (got $code)" test "$code" = "200"
  check "GET /api/status success:true" json_has_success_true "$status_file"

  local models_file="$TMP_DIR/models_no_auth.json"
  code="$(curl_save "$models_file" "${PUBLIC_URL}/v1/models")"
  check "GET /v1/models without token is rejected (HTTP 401, got $code)" test "$code" = "401"

  local openai_models_file="$TMP_DIR/openai_models_no_auth.json"
  code="$(curl_save "$openai_models_file" "${PUBLIC_URL}/openai/models")"
  check "GET /openai/models without token is rejected (HTTP 401, got $code)" test "$code" = "401"
}

run_auth() {
  echo ""
  echo "=== Phase: auth ==="

  if [[ -z "$USER_TOKEN" ]]; then
    echo "[SKIP] auth phase: --token not provided"
    return 0
  fi

  local bad_file="$TMP_DIR/models_bad_token.json"
  local code
  code="$(curl_save "$bad_file" -H "Authorization: Bearer sk-invalid-token-for-verify" "${PUBLIC_URL}/v1/models")"
  check "invalid token rejected on /v1/models (HTTP 401, got $code)" test "$code" = "401"

  local ok_file="$TMP_DIR/models_ok.json"
  code="$(curl_save "$ok_file" -H "Authorization: Bearer ${USER_TOKEN}" "${PUBLIC_URL}/v1/models")"
  check "valid token accepted on /v1/models (HTTP 200, got $code)" test "$code" = "200"
}

run_relay() {
  echo ""
  echo "=== Phase: relay ==="

  if [[ -z "$USER_TOKEN" ]]; then
    echo "[SKIP] relay phase: --token not provided"
    return 0
  fi

  local models_file="$TMP_DIR/models_relay.json"
  local code
  code="$(curl_save "$models_file" -H "Authorization: Bearer ${USER_TOKEN}" "${PUBLIC_URL}/v1/models")"
  check "GET /v1/models HTTP 200 (got $code)" test "$code" = "200"

  if command -v jq >/dev/null 2>&1; then
    check "model list contains ${MODEL}" jq -e --arg m "$MODEL" '[.data[]?.id] | index($m) != null' "$models_file" >/dev/null
  else
    check "model list contains ${MODEL}" grep -q "\"id\"[[:space:]]*:[[:space:]]*\"${MODEL}\"" "$models_file"
  fi

  local chat_file="$TMP_DIR/chat_completions.json"
  local chat_body='{"model":"'"${MODEL}"'","messages":[{"role":"user","content":"hi"}],"max_tokens":8}'
  code="$(curl_save "$chat_file" -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$chat_body" \
    "${PUBLIC_URL}/v1/chat/completions")"
  # Codex channel does not support chat/completions unless chat->responses policy is enabled.
  # Production Codex CLI uses /v1/responses only; we expect failure or non-codex routing here.
  if [[ "$code" == "200" ]]; then
    echo "[WARN] POST /v1/chat/completions returned 200 — chat->responses policy may be enabled; Codex CLI should still use /v1/responses"
  else
    check "POST /v1/chat/completions not primary Codex path (HTTP != 200, got $code)" test "$code" != "200"
  fi
}

run_codex() {
  echo ""
  echo "=== Phase: codex ==="

  if [[ -z "$USER_TOKEN" ]]; then
    echo "[FAIL] codex phase requires --token" >&2
    FAIL=1
    return 0
  fi

  local sync_file="$TMP_DIR/responses_sync.json"
  local sync_body='{"model":"'"${MODEL}"'","input":[{"role":"user","content":"Reply with exactly: ok"}],"stream":false}'
  local code
  code="$(curl_save "$sync_file" -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$sync_body" \
    "${PUBLIC_URL}/v1/responses")"
  check "POST /v1/responses sync HTTP 200 (got $code)" test "$code" = "200"

  if command -v jq >/dev/null 2>&1; then
    check "POST /v1/responses sync has response id" jq -e '.id != null and .id != ""' "$sync_file" >/dev/null
    local err_type
    err_type="$(jq -r '.error.type // empty' "$sync_file" 2>/dev/null || true)"
    check "POST /v1/responses sync no error.type" test -z "$err_type"
  else
    check "POST /v1/responses sync body non-empty" test -s "$sync_file"
    check "POST /v1/responses sync no error field" ! grep -q '"error"[[:space:]]*:[[:space:]]*{' "$sync_file"
  fi

  local stream_file="$TMP_DIR/responses_stream.txt"
  local stream_body='{"model":"'"${MODEL}"'","input":[{"role":"user","content":"Reply with exactly: ok"}],"stream":true}'
  code="$(curl_save "$stream_file" -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d "$stream_body" \
    "${PUBLIC_URL}/v1/responses")"
  check "POST /v1/responses stream HTTP 200 (got $code)" test "$code" = "200"
  check "POST /v1/responses stream returns SSE data lines" grep -q '^data:' "$stream_file"
  check "POST /v1/responses stream not upstream error body" ! grep -qi '"type"[[:space:]]*:[[:space:]]*"invalid_request_error"' "$stream_file"

  local openai_sync_file="$TMP_DIR/openai_responses_sync.json"
  code="$(curl_save "$openai_sync_file" -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "$sync_body" \
    "${PUBLIC_URL}/openai/responses")"
  check "POST /openai/responses sync HTTP 200 (got $code)" test "$code" = "200"

  local openai_stream_file="$TMP_DIR/openai_responses_stream.txt"
  code="$(curl_save "$openai_stream_file" -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d "$stream_body" \
    "${PUBLIC_URL}/openai/responses")"
  check "POST /openai/responses stream HTTP 200 (got $code)" test "$code" = "200"
  check "POST /openai/responses stream returns SSE data lines" grep -q '^data:' "$openai_stream_file"
}

echo "=== Codex relay verification ==="
echo "URL:   $PUBLIC_URL"
echo "Model: $MODEL"
echo "Phase: $PHASE"

phase_enabled infra && run_infra
phase_enabled auth && run_auth
phase_enabled relay && run_relay
phase_enabled codex && run_codex

echo ""
if [[ "$FAIL" -eq 0 ]]; then
  echo "=== All checks passed ==="
  exit 0
fi

echo "=== Some checks failed ==="
echo "See docs/installation/codex-relay-production.md for remediation."
exit 1
