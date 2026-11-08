#!/bin/sh
# One-time signer setup for @jdl. The Go backend does everything:
# create signer -> sign with your mnemonic (local only) -> register ->
# print approval URL -> poll -> save SIGNER_UUID to .env
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"
if [ ! -f .env ]; then
  echo "copy .env.example to .env and fill it in first" >&2
  exit 1
fi
chmod 600 .env 2>/dev/null || true
(cd backend && go run . -signer-setup)
