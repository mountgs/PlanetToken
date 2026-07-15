#!/usr/bin/env bash
# Backup PostgreSQL for docker-compose.prod.yml deployments.
#
# Usage:
#   ./scripts/backup-db.sh
#   ./scripts/backup-db.sh --label v1.2.3
#
# Requires: docker compose, running postgres service, .env.production

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env.production}"
BACKUP_DIR="${BACKUP_DIR:-${ROOT_DIR}/backups}"
LABEL=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --label)
      LABEL="$2"
      shift 2
      ;;
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --env-file)
      ENV_FILE="$2"
      shift 2
      ;;
    --backup-dir)
      BACKUP_DIR="$2"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 [--label VERSION] [--compose-file FILE] [--env-file FILE] [--backup-dir DIR]"
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

cd "$ROOT_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE — copy from .env.production.example first." >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

: "${POSTGRES_USER:?POSTGRES_USER not set in $ENV_FILE}"
: "${POSTGRES_DB:?POSTGRES_DB not set in $ENV_FILE}"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
IMAGE_TAG="${NEW_API_IMAGE:-calciumion/new-api:latest}"
IMAGE_TAG="${IMAGE_TAG##*:}"
if [[ -n "$LABEL" ]]; then
  SAFE_LABEL="$(echo "$LABEL" | tr -cs '[:alnum:]._-' '_')"
  BASENAME="new-api_${POSTGRES_DB}_${TIMESTAMP}_${SAFE_LABEL}"
else
  BASENAME="new-api_${POSTGRES_DB}_${TIMESTAMP}_${IMAGE_TAG}"
fi

mkdir -p "$BACKUP_DIR"
DUMP_FILE="${BACKUP_DIR}/${BASENAME}.sql.gz"
META_FILE="${BACKUP_DIR}/${BASENAME}.meta.json"

echo "=== Backing up PostgreSQL ==="
echo "Database : ${POSTGRES_DB}"
echo "Output   : ${DUMP_FILE}"

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-acl \
  | gzip -9 > "$DUMP_FILE"

cat > "$META_FILE" <<EOF
{
  "created_at": "${TIMESTAMP}",
  "postgres_db": "${POSTGRES_DB}",
  "postgres_user": "${POSTGRES_USER}",
  "image": "${NEW_API_IMAGE:-calciumion/new-api:latest}",
  "label": "${LABEL}",
  "dump_file": "$(basename "$DUMP_FILE")",
  "host": "$(hostname -f 2>/dev/null || hostname)"
}
EOF

echo "=== Backup complete ==="
echo "Dump : $DUMP_FILE"
echo "Meta : $META_FILE"
echo ""
echo "Retention tip: keep at least one backup before each upgrade."
