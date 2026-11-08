# fc-tui — Farcaster feed + posting TUI (Go backend + Bubble Tea, Quickshell bar paused)

Backend uses ~10MB RAM, 12MB binary. Good for 8GB laptops.
TUI runs anywhere Go runs: Mac, Linux, tmux/zellij/herdr. Quickshell bar is paused.

## Setup (both machines)

1. Copy your key:
```
cp .env.example .env
# edit .env: NEYNAR_API_KEY needs no quotes,
# FARCASTER_DEVELOPER_MNEMONIC MUST stay quoted (12 words, shell sources this file)
```

2. One-time signer (posts as your fid, Neynar sponsors gas):
```
./signer-setup.sh
# open the printed URL on your phone, approve, press ENTER
# saves SIGNER_UUID to .env
```

3. Test backend (Mac or Linux):
```
./refresh.sh
# prints 15 casts, writes feed.json
```

## Run TUI (Mac, Linux, tmux/zellij/herdr)

No GUI needed — the Bubble Tea TUI replaces the Quickshell bar anywhere:

```
cd tui && go run .        # or: go build -o fc-tui . && ./fc-tui
```

Keys: `j/k` move · `enter` cast detail · `i` write · `R` reply · `Q` quote · `r` refresh · `q` quit.
Replies target the highlighted cast as parent, quotes embed it. Esc cancels.
New casts pop a `herdr notification show` alert when run inside herdr.
Auto-refreshes every 2 min, same as the bar.

## Run UI (Linux box with Quickshell)

`-c` looks in `~/.config/quickshell/`, not the current folder.
Pick one:

Option A — install as named config:
```
mkdir -p ~/.config/quickshell
ln -sfn "$PWD" ~/.config/quickshell/fc-tui
quickshell -c fc-tui
```

Option B — run by path, no install:
```
quickshell -p "$PWD/shell.qml"
```

Bar shows following-feed for @jdl (fid 10695).
Auto-refreshes every 2 min via `refresh.sh`.

Change feed: `FEED_FID=your_fid ./refresh.sh`

## Posting (quick write)

One-time setup — the backend signs locally with your recovery phrase,
registers the signer with Neynar, and prints an approval link:
```
# 1. put your Warpcast recovery phrase in .env (12 words, stays on this laptop)
# 2. then:
./signer-setup.sh
# open the printed URL in the phone app, approve, press ENTER
# saves SIGNER_UUID to .env
```

Security notes:
- `FARCASTER_DEVELOPER_MNEMONIC` is your master recovery phrase. Anyone with
  it owns the account. Keep `.env` at chmod 600, never commit it (gitignored),
  never paste it in chat. You can delete that line from `.env` after setup —
  only `SIGNER_UUID` (revocable) is needed to post.
- Sponsorship defaults to Neynar (gas covered, costs credits). The approval
  screen shows who pays.

Then post from terminal or from the bar's composer box:
```
./post.sh "hello farcaster"
```

The bar has a text box + Post button at the top. Status shows Posted! or red error.

## Files

- `backend/main.go` — stdlib-only Neynar fetch, no SDK
- `refresh.sh` — loads .env, runs backend
- `shell.qml` — PanelWindow + FileView watcher
- `feed.json` — generated, gitignored
