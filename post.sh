#!/bin/sh
# Quick post: ./post.sh "hello farcaster"
# Reply: ./post.sh --reply HASH FID "your reply"
# Quote: ./post.sh --quote HASH FID "your take"
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
if [ $# -eq 0 ]; then
  echo 'usage: ./post.sh "your text" | ./post.sh --reply HASH FID "text" | ./post.sh --quote HASH FID "text"' >&2
  exit 1
fi
MODE=""
if [ "$1" = "--reply" ] || [ "$1" = "--quote" ]; then
  MODE="$1"; shift
fi
if [ -n "$MODE" ]; then
  if [ $# -lt 3 ]; then
    echo "usage: ./post.sh $MODE HASH FID \"text\"" >&2
    exit 1
  fi
  HASH="$1"; FID="$2"; shift 2
  if [ "$MODE" = "--reply" ]; then
    (cd backend && go run . -post "$*" -parent "$HASH" -parent-fid "$FID") && ./refresh.sh >/dev/null 2>&1
  else
    (cd backend && go run . -post "$*" -quote-hash "$HASH" -quote-fid "$FID") && ./refresh.sh >/dev/null 2>&1
  fi
else
  (cd backend && go run . -post "$*") && ./refresh.sh >/dev/null 2>&1
fi
