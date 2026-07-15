#!/usr/bin/env bash
# First-time production deployment on the server (Baota + Docker Compose).
#
# Run on the server inside the deployment directory, e.g. /www/wwwroot/new-api
#
# Usage:
#   ./scripts/deploy-prod.sh
#   ./scripts/deploy-prod.sh --skip-pull

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env.production}"
SKIP_PULL=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-pull) SKIP_PULL=true; shift ;;
    --compose-file) COMPOSE_FILE="$2"; shift 2 ;;
    --env-file) ENV_FILE="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--skip-pull] [--compose-file FILE] [--env-file FILE]"
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

cd "$ROOT_DIR"

echo "=== New API production deploy ==="
echo "Directory: $ROOT_DIR"

if [[ ! -f "$COMPOSE_FILE" ]]; then
  echo "Missing $COMPOSE_FILE" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE" >&2
  echo "Run: cp .env.production.example .env.production && edit secrets" >&2
  exit 1
fi

if grep -q 'CHANGE_ME' "$ENV_FILE"; then
  echo "ERROR: $ENV_FILE still contains CHANGE_ME placeholders." >&2
  exit 1
fi

chmod 600 "$ENV_FILE"
mkdir -p data logs backups
chmod +x scripts/backup-db.sh scripts/restore-db.sh scripts/verify-deployment.sh 2>/dev/null || true

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: docker not found. Install Docker via Baota first." >&2
  exit 1
fi

COMPOSE=(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE")

if [[ "$SKIP_PULL" != true ]]; then
  echo "=== Pulling images ==="
  "${COMPOSE[@]}" pull
fi

echo "=== Starting services ==="
"${COMPOSE[@]}" up -d

echo "=== Waiting for health ==="
for i in $(seq 1 30); do
  if curl -sf http://127.0.0.1:3000/api/status | grep -q '"success":true'; then
    echo "Health check passed."
    break
  fi
  if [[ "$i" -eq 30 ]]; then
    echo "ERROR: health check failed after 30 attempts." >&2
    "${COMPOSE[@]}" ps
    "${COMPOSE[@]}" logs new-api --tail 50
    exit 1
  fi
  sleep 2
done

echo ""
echo "=== Deploy complete ==="
echo "Local check: curl -s http://127.0.0.1:3000/api/status"
echo "Next: configure Baota nginx + SSL (see docs/installation/production.md)"
echo "Verify: ./scripts/verify-deployment.sh"
