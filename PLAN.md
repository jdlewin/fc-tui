# Plan — Farcaster TUI

Goal: Read and post Farcaster from the Bubble Tea TUI (Mac, Linux, tmux)

## Now (TUI)

- [x] Backend refresh prints 15 casts on Mac — verified 08 Nov, 15/15
- [x] TUI run-commit-polish: commit tui/ + PLAN + README a53fd18, vet clean, 11 tests pass, headless smoke expected tty fail
- [x] Post from the TUI and confirm round-trip — live 08 Nov as donniecast, cast ab0b6b6e, success true
- [x] Harden composer path 08 Nov — 3 submit tests (15 pass), backend empty-cast guard, dotenv quote fix, README TUI-first

## Paused (Quickshell)

- [ ] Bar runs on Linux box via quickshell — paused per jd 08 Nov
- [ ] Post from the bar and confirm round-trip — paused per jd 08 Nov

## Later

- [x] Review with a human — eyeballed 08 Nov, list nav detail composer-cancel refresh all as expected
