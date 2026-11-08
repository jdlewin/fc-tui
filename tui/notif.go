// Notifications poll: mentions, replies, quotes aimed at you.
// Feed refresh sees the timeline; this sees people talking to you.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const notifEvery = 5 * time.Minute

type notif struct {
	Type   string
	Hash   string
	Author string
	Text   string
}

type notifsLoadedMsg struct{ items []notif }
type notifErrMsg struct{ err error }
type notifTickMsg struct{}

func scheduleNotif() tea.Cmd {
	return tea.Tick(notifEvery, func(time.Time) tea.Msg { return notifTickMsg{} })
}

func fetchNotifsCmd(key, fid string) tea.Cmd {
	return func() tea.Msg {
		items, err := fetchNotifs(key, fid)
		if err != nil {
			return notifErrMsg{err}
		}
		return notifsLoadedMsg{items}
	}
}

func fetchNotifs(key, fid string) ([]notif, error) {
	url := "https://api.neynar.com/v2/farcaster/notifications/?fid=" + fid + "&type=mentions,replies,quotes&limit=20"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", key)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("neynar %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Notifications []struct {
			Type string `json:"type"`
			Cast *struct {
				Hash   string `json:"hash"`
				Text   string `json:"text"`
				Author struct {
					Username string `json:"username"`
				} `json:"author"`
			} `json:"cast"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	var items []notif
	for _, n := range out.Notifications {
		if n.Cast == nil {
			continue
		}
		t := strings.ToLower(n.Type)
		switch t {
		case "mention", "mentions", "reply", "replies", "quote", "quotes":
			items = append(items, notif{Type: t, Hash: n.Cast.Hash, Author: n.Cast.Author.Username, Text: n.Cast.Text})
		}
	}
	return items, nil
}

func notifTitle(n notif) string {
	switch {
	case strings.HasPrefix(n.Type, "mention"):
		return "Farcaster — you were mentioned"
	case strings.HasPrefix(n.Type, "repl"):
		return "Farcaster — reply to you"
	default:
		return "Farcaster — your cast was quoted"
	}
}
