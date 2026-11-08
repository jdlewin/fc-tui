# fc-tui — Farcaster feed for Quickshell (Go backend + QML bar)

Backend uses ~9MB RAM, 6.5MB binary. Good for 8GB laptops.
Quickshell UI runs on Linux only (Wayland). Backend runs anywhere Go runs.

## Setup (both machines)

1. Copy your key:
```
cp .env.example .env
# edit .env, put NEYNAR_API_KEY=xxx (no quotes)
```

2. Test backend (Mac or Linux):
```
./refresh.sh
# prints 15 casts, writes feed.json
```

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

Bar shows following-feed for fid 3 (dwr.eth demo).
Auto-refreshes every 2 min via `refresh.sh`.

Change feed: `FEED_FID=your_fid ./refresh.sh`

## Files

- `backend/main.go` — stdlib-only Neynar fetch, no SDK
- `refresh.sh` — loads .env, runs backend
- `shell.qml` — PanelWindow + FileView watcher
- `feed.json` — generated, gitignored
