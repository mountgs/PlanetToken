#!/usr/bin/env bash
# Restore PostgreSQL from a backup created by scripts/backup-db.sh
#
# Usage:
#   ./scripts/restore-db.sh backups/new-api_new-api_20260101T120000Z_latest.sql.gz
#   ./scripts/restore-db.sh --yes backups/....sql.gz
#
# WARNING: This replaces all data in the target database.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env.production}"
ASSUME_YES=false
DUMP_FILE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --yes|-y)
      ASSUME_YES=true
      shift
      ;;
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --env-file)
      ENV_FILE="$2"
      shift 2
      ;;
    -h|--help)
      echo "Usage: $0 [--yes] <path-to-backup.sql.gz>"
      exit 0
      ;;
    *)
      if [[ -z "$DUMP_FILE" ]]; then
        DUMP_FILE="$1"
      else
        echo "Unexpected argument: $1" >&2
        exit 1
      fi
      shift
      ;;
  esac
done

if [[ -z "$DUMP_FILE" ]]; then
  echo "Provide path to .sql.gz backup file." >&2
  exit 1
fi

if [[ ! -f "$DUMP_FILE" ]]; then
  echo "Backup file not found: $DUMP_FILE" >&2
  exit 1
fi

cd "$ROOT_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

: "${POSTGRES_USER:?POSTGRES_USER not set in $ENV_FILE}"
: "${POSTGRES_DB:?POSTGRES_DB not set in $ENV_FILE}"

META_FILE="${DUMP_FILE%.sql.gz}.meta.json"
if [[ -f "$META_FILE" ]]; then
  echo "Backup metadata:"
  cat "$META_FILE"
  echo ""
fi

echo "=== RESTORE WARNING ==="
echo "This will DROP and recreate database: ${POSTGRES_DB}"
echo "Backup file: ${DUMP_FILE}"
echo ""

if [[ "$ASSUME_YES" != true ]]; then
  read -r -p "Type RESTORE to continue: " CONFIRM
  if [[ "$CONFIRM" != "RESTORE" ]]; then
    echo "Aborted."
    exit 1
  fi
fi

echo "Stopping new-api to release DB connections..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" stop new-api

echo "Restoring database..."
gunzip -c "$DUMP_FILE" | docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$POSTGRES_USER" -d postgres -v ON_ERROR_STOP=1 <<SQL
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${POSTGRES_DB}' AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS "${POSTGRES_DB}";
CREATE DATABASE "${POSTGRES_DB}" OWNER "${POSTGRES_USER}";
SQL

gunzip -c "$DUMP_FILE" | docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
  psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1

echo "Starting new-api..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d new-api

echo "=== Restore complete ==="
echo "Verify: curl -s http://127.0.0.1:3000/api/status"
