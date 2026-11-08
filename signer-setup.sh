#!/bin/sh
# One-time signer setup for @jdl. Creates ONE signer, you approve it in Warpcast/Farcaster app.
# Costs an onchain tx on approve (Neynar may sponsor — check the approval screen).
# Reuses existing SIGNER_UUID if already approved.
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

if [ -n "${SIGNER_UUID:-}" ]; then
  echo "checking existing SIGNER_UUID=$SIGNER_UUID"
  st=$(curl -s "https://api.neynar.com/v2/farcaster/signer/?signer_uuid=$SIGNER_UUID" -H "x-api-key: $NEYNAR_API_KEY")
  echo "$st" | head -c 400; echo
  case "$st" in
    *'"status":"approved"'*) echo "already approved — nothing to do"; exit 0;;
  esac
  echo "(existing signer not approved — creating a fresh one)"
fi

echo "creating signer…"
resp=$(curl -s -X POST "https://api.neynar.com/v2/farcaster/signer/" -H "x-api-key: $NEYNAR_API_KEY" -H "Content-Type: application/json" -d '{}')
echo "$resp"
uuid=$(echo "$resp" | python3 -c "import sys,json;print(json.load(sys.stdin).get('signer_uuid',''))")
url=$(echo "$resp" | python3 -c "import sys,json;print(json.load(sys.stdin).get('signer_approval_url',''))")
if [ -z "$uuid" ]; then echo "create failed" >&2; exit 1; fi

echo
echo "1. Open this URL on your phone (Warpcast/Farcaster app):"
echo "   $url"
echo "2. Approve the signer. Then press ENTER here."
read -r _

for i in 1 2 3 4 5 6; do
  st=$(curl -s "https://api.neynar.com/v2/farcaster/signer/?signer_uuid=$uuid" -H "x-api-key: $NEYNAR_API_KEY")
  echo "poll $i: $(echo "$st" | head -c 200)"
  case "$st" in
    *'"status":"approved"'*)
      echo "approved!"
      if grep -q "^SIGNER_UUID=" .env 2>/dev/null; then
        sed -i.bak "s/^SIGNER_UUID=.*/SIGNER_UUID=$uuid/" .env
      else
        echo "SIGNER_UUID=$uuid" >> .env
      fi
      echo "saved SIGNER_UUID to .env"
      exit 0;;
  esac
  sleep 10
done
echo "not approved yet — re-run after approving. UUID: $uuid" >&2
echo "URL: $url" >&2
exit 1
