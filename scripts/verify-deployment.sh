#!/usr/bin/env bash
# Post-deployment verification (run on server after deploy + nginx setup).
#
# Usage:
#   ./scripts/verify-deployment.sh
#   ./scripts/verify-deployment.sh --url https://api.example.com

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env.production}"
PUBLIC_URL=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url) PUBLIC_URL="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--url https://your-domain.com]"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

cd "$ROOT_DIR"
COMPOSE=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")
FAIL=0

env_no_change_me() {
  if grep -q 'CHANGE_ME' "$ENV_FILE"; then
    echo "  placeholder lines:" >&2
    grep -n 'CHANGE_ME' "$ENV_FILE" >&2 || true
    return 1
  fi
  return 0
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

echo "=== Deployment verification ==="

check "docker compose available" command -v docker
check ".env.production exists" test -f "$ENV_FILE"
check "no CHANGE_ME in env" env_no_change_me
check "new-api container running" "${COMPOSE[@]}" ps --status running --services | grep -qx new-api
check "postgres container running" "${COMPOSE[@]}" ps --status running --services | grep -qx postgres
check "redis container running" "${COMPOSE[@]}" ps --status running --services | grep -qx redis
check "local API status" curl -sf http://127.0.0.1:3000/api/status | grep -q '"success":true'
check "port 3000 bound to localhost" ss -tln 2>/dev/null | grep -q '127.0.0.1:3000' || netstat -tln 2>/dev/null | grep -q '127.0.0.1:3000'

if [[ -n "$PUBLIC_URL" ]]; then
  check "public HTTPS status" curl -sf "${PUBLIC_URL%/}/api/status" | grep -q '"success":true'
fi

echo ""
if [[ "$FAIL" -eq 0 ]]; then
  echo "=== All checks passed ==="
  exit 0
fi

echo "=== Some checks failed ==="
exit 1
