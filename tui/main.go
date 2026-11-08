// fc-tui — Farcaster feed TUI. Runs anywhere Go runs: macOS, Linux,
// tmux/zellij/herdr. No GUI, no Qt, no Wayland.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	loadDotEnv(".env")
	loadDotEnv("../.env")

	key := os.Getenv("NEYNAR_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "missing NEYNAR_API_KEY in env or .env")
		os.Exit(1)
	}
	fid := os.Getenv("FEED_FID")
	if fid == "" {
		fid = "10695" // @jdl
	}
	feedPath := firstExisting([]string{"feed.json", "../feed.json"}, "feed.json")

	initial, _ := loadFeedFile(feedPath) // empty is fine; network refresh fills in

	p := tea.NewProgram(NewModel(key, fid, feedPath, initial), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}
