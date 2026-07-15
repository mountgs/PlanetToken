#!/usr/bin/env bash
# Copy deployment bundle from dev machine to server via rsync/scp.
#
# Usage:
#   SERVER=203.0.113.10 USER=root SSH_KEY=~/.ssh/id_rsa ./scripts/push-to-server.sh
#   REMOTE_DIR=/www/wwwroot/new-api ./scripts/push-to-server.sh
#
# Requires: ssh, rsync (or scp fallback)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="${SERVER:?Set SERVER=your.server.ip}"
USER="${USER:-root}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_DIR="${REMOTE_DIR:-/www/wwwroot/new-api}"
SSH_OPTS=(-o StrictHostKeyChecking=accept-new)

if [[ -n "$SSH_KEY" ]]; then
  SSH_OPTS+=(-i "$SSH_KEY")
fi

REMOTE="${USER}@${SERVER}"
RSYNC_SSH="ssh ${SSH_OPTS[*]}"

FILES=(
  docker-compose.prod.yml
  .env.production.example
  scripts/deploy-prod.sh
  scripts/verify-deployment.sh
  scripts/backup-db.sh
  scripts/restore-db.sh
  deploy/nginx/new-api.conf.example
  docs/installation/production.md
  docs/installation/operations.md
)

echo "=== Push deploy bundle to ${REMOTE}:${REMOTE_DIR} ==="

ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p ${REMOTE_DIR}/{scripts,deploy/nginx,docs/installation,backups,data,logs}"

if command -v rsync >/dev/null 2>&1; then
  for f in "${FILES[@]}"; do
    src="${ROOT_DIR}/${f}"
    if [[ ! -f "$src" ]]; then
      echo "Skip missing: $f"
      continue
    fi
    rsync -avz -e "$RSYNC_SSH" "$src" "${REMOTE}:${REMOTE_DIR}/${f}"
  done
else
  for f in "${FILES[@]}"; do
    src="${ROOT_DIR}/${f}"
    [[ -f "$src" ]] || continue
    scp "${SSH_OPTS[@]}" "$src" "${REMOTE}:${REMOTE_DIR}/${f}"
  done
fi

echo ""
echo "=== Push complete ==="
echo "Next on server:"
echo "  ssh ${SSH_OPTS[*]} ${REMOTE}"
echo "  cd ${REMOTE_DIR}"
echo "  cp .env.production.example .env.production && vim .env.production"
echo "  ./scripts/deploy-prod.sh"
