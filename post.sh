#!/bin/sh
# Quick post: ./post.sh "hello farcaster"
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
if [ $# -eq 0 ]; then
  echo 'usage: ./post.sh "your text"' >&2
  exit 1
fi
(cd backend && go run . -post "$*") && ./refresh.sh >/dev/null 2>&1
