#!/bin/sh
# One-time signer setup for @jdl.
# The approval URL comes from the Neynar dev portal (no mnemonic needed on this laptop).
# Usage: ./signer-setup.sh [SIGNER_UUID]
#    or: SIGNER_UUID=xxx ./signer-setup.sh
set -u
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR"
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi
if [ -z "${NEYNAR_API_KEY:-}" ]; then
  echo "missing NEYNAR_API_KEY in .env" >&2
  exit 1
fi

UUID="${1:-${SIGNER_UUID:-}}"
if [ -z "$UUID" ]; then
  echo "No signer yet. Do this once in the browser:"
  echo "  1. Open https://dev.neynar.com and pick your app (same app as your API key)"
  echo "  2. Find Signers -> create/add a signer for @jdl (fid 10695)"
  echo "  3. Approve it in the Warpcast/Farcaster phone app"
  echo "  4. Copy the signer UUID, then run:"
  echo "       ./signer-setup.sh PASTE-UUID-HERE"
  exit 0
fi

echo "checking signer $UUID…"
st=$(curl -sS "https://api.neynar.com/v2/farcaster/signer/?signer_uuid=$UUID" -H "x-api-key: $NEYNAR_API_KEY" 2>&1)
echo "$st" | head -c 500; echo
case "$st" in
  *'"status":"approved"'*)
    echo "approved!"
    if grep -q "^SIGNER_UUID=" .env 2>/dev/null; then
      sed -i.bak "s/^SIGNER_UUID=.*/SIGNER_UUID=$UUID/" .env
    else
      echo "SIGNER_UUID=$UUID" >> .env
    fi
    echo "saved SIGNER_UUID to .env — try: ./post.sh \"hello from my bar\""
    exit 0;;
  *'"status":"pending_approval"'*)
    echo "still pending — approve it in the phone app, then re-run this script" >&2
    exit 1;;
  *)
    echo "signer not usable (see response above)" >&2
    exit 1;;
esac
