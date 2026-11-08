// herdr notification integration: new casts surface as OS-level
// notifications through the multiplexer, not as in-TUI chrome.
package main

import (
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type notifyDoneMsg struct{}

func notifyCmd(title, body string) tea.Cmd {
	return notifyTypedCmd(title, body, "none", "")
}

// notifyTypedCmd shows a herdr note and optionally plays a Mac system sound.
// sound is a herdr chime (none/done/request); macSound names a file in
// /System/Library/Sounds without extension ("" = silent). afplay missing
// (Linux) means visual only.
func notifyTypedCmd(title, body, sound, macSound string) tea.Cmd {
	return func() tea.Msg {
		if macSound != "" {
			if player, err := exec.LookPath("afplay"); err == nil {
				_ = exec.Command(player, "/System/Library/Sounds/"+macSound+".aiff").Run()
			}
		}
		herdr, err := exec.LookPath("herdr")
		if err != nil {
			return notifyDoneMsg{}
		}
		_ = exec.Command(herdr, "notification", "show", title, "--body", body, "--sound", sound).Run()
		return notifyDoneMsg{}
	}
}

// macSoundFor maps a Farcaster notification type to jd's picks.
func macSoundFor(notifType string) (herdrSound, mac string) {
	switch notifType {
	case "mention", "mentions":
		return "request", "Sosumi"
	case "reply", "replies", "quote", "quotes":
		return "done", "Frog"
	default:
		return "none", ""
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
