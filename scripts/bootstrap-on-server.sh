#!/usr/bin/env bash
# One-shot bootstrap for production deploy ON THE SERVER.
# Run after files are in /www/wwwroot/new-api (git clone or Baota upload).
#
# Usage (on server as root):
#   cd /www/wwwroot/new-api
#   chmod +x scripts/bootstrap-on-server.sh
#   ./scripts/bootstrap-on-server.sh
#
# Optional:
#   DOMAIN=api.example.com ./scripts/bootstrap-on-server.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE=".env.production"
COMPOSE_FILE="docker-compose.prod.yml"

echo "=== New API server bootstrap ==="
echo "Directory: $ROOT_DIR"

if [[ $EUID -ne 0 ]]; then
  echo "WARN: not running as root; Docker may fail." >&2
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "ERROR: Docker not installed." >&2
  echo "Install via Baota: Docker -> Install, then re-run this script." >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "ERROR: docker compose plugin not found." >&2
  exit 1
fi

gen_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    head -c 32 /dev/urandom | xxd -p -c 64
  fi
}

if [[ ! -f "$ENV_FILE" ]]; then
  echo "=== Generating $ENV_FILE ==="
  PG_PASS="$(gen_secret)"
  REDIS_PASS="$(gen_secret)"
  SESSION="$(gen_secret)"
  CRYPTO="$(gen_secret)"

  cat > "$ENV_FILE" <<EOF
POSTGRES_USER=newapi
POSTGRES_PASSWORD=${PG_PASS}
POSTGRES_DB=new-api

REDIS_PASSWORD=${REDIS_PASS}

SESSION_SECRET=${SESSION}
CRYPTO_SECRET=${CRYPTO}

TZ=Asia/Shanghai
ERROR_LOG_ENABLED=true
BATCH_UPDATE_ENABLED=true
NODE_NAME=new-api-prod-1

NEW_API_IMAGE=calciumion/new-api:latest
EOF
  echo "Created $ENV_FILE with random secrets."
else
  if grep -q 'CHANGE_ME' "$ENV_FILE"; then
    echo "ERROR: Edit $ENV_FILE and replace CHANGE_ME values first." >&2
    exit 1
  fi
  echo "Using existing $ENV_FILE"
fi

chmod 600 "$ENV_FILE"
chmod +x scripts/*.sh 2>/dev/null || true

./scripts/deploy-prod.sh

echo ""
echo "=== Bootstrap finished ==="
echo ""
echo "Local API: curl -s http://127.0.0.1:3000/api/status"
echo ""
echo "--- Next: Baota nginx (required for browser access) ---"
echo "1. Baota -> Website -> Add site"
if [[ -n "${DOMAIN:-}" ]]; then
  echo "   Domain: ${DOMAIN}"
else
  echo "   Domain: your server IP or bound domain"
fi
echo "2. Site config -> reverse proxy to http://127.0.0.1:3000"
echo "   proxy_buffering off; proxy_read_timeout 600s;"
echo "3. SSL -> Let's Encrypt (if you have a domain)"
echo "4. Verify: ./scripts/verify-deployment.sh --url https://YOUR_DOMAIN"
echo ""
echo "Baota panel: usually https://SERVER_IP:8888 (check your provider)"
echo "Ops manual: docs/installation/operations.md"
