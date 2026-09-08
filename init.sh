#!/bin/bash
set -e

echo "=== Harness Initialization ==="

temporary_web_dist=false
if [ ! -f web/dist/index.html ]; then
  mkdir -p web/dist
  printf '%s\n' '<!doctype html><title>test placeholder</title>' > web/dist/index.html
  temporary_web_dist=true
fi

cleanup() {
  if [ "$temporary_web_dist" = true ]; then
    rm -f web/dist/index.html
    rmdir web/dist 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "=== go test ./... ==="
go test ./...

echo "=== cd web && bun run typecheck ==="
(cd web && bun run typecheck)

echo "=== Verification Complete ==="
echo ""
echo "Next steps:"
echo "1. Read feature_list.json to see current feature state"
echo "2. Pick ONE unfinished feature to work on"
echo "3. Implement only that feature"
echo "4. Re-run verification before claiming done"
