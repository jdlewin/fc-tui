#!/bin/sh
# Refresh feed.json using your .env key. Low-memory: one fetch, 15 casts.
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
if [ -z "${NEYNAR_API_KEY:-}" ]; then
  echo "missing NEYNAR_API_KEY (put it in .env)" >&2
  exit 1
fi
cd "$SCRIPT_DIR/backend" && go run .
