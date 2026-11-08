// herdr notification integration: new casts surface as OS-level
// notifications through the multiplexer, not as in-TUI chrome.
package main

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type notifyDoneMsg struct{}

func notifyCmd(title, body string) tea.Cmd {
	return func() tea.Msg {
		herdr, err := exec.LookPath("herdr")
		if err != nil {
			return notifyDoneMsg{}
		}
		cmd := exec.Command(herdr, "notification", "show", title, "--body", body)
		_ = cmd.Run()
		return notifyDoneMsg{}
	}
}

func newCastNotification(casts []Cast) (string, string) {
	if len(casts) == 1 {
		c := casts[0]
		return "Farcaster — 1 new cast", "@" + displayName(c) + ": " + oneLine(c.Text, 120)
	}
	first := casts[0]
	return "Farcaster — new casts", "@" + displayName(first) + " and more: " + oneLine(first.Text, 100)
}

func displayName(c Cast) string {
	if c.Author.Username != "" {
		return c.Author.Username
	}
	return "unknown"
}
