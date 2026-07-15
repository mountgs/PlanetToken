#!/usr/bin/env bash
# Create deploy-bundle.tar.gz for Baota file upload.
# Run on dev machine from repo root:
#   ./scripts/make-deploy-bundle.sh

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
OUT="deploy-bundle.tar.gz"

tar czf "$OUT" \
  docker-compose.prod.yml \
  .env.production.example \
  scripts/bootstrap-on-server.sh \
  scripts/deploy-prod.sh \
  scripts/verify-deployment.sh \
  scripts/backup-db.sh \
  scripts/restore-db.sh \
  deploy/nginx/new-api.conf.example \
  docs/installation/production.md \
  docs/installation/operations.md

echo "Created $ROOT_DIR/$OUT"
echo "Upload to server /www/wwwroot/new-api/ and run:"
echo "  tar xzf deploy-bundle.tar.gz && chmod +x scripts/*.sh && ./scripts/bootstrap-on-server.sh"
