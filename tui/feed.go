// Feed data + Neynar access. Stdlib only (same shape as backend/main.go).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Author struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Fid         int    `json:"fid"`
}

type Cast struct {
	Hash      string `json:"hash"`
	Text      string `json:"text"`
	Author    Author `json:"author"`
	Timestamp string `json:"timestamp"`
}

type persistedFeed struct {
	Casts []Cast `json:"casts"`
	Next  any    `json:"next"`
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if _, ok := os.LookupEnv(k); !ok {
			os.Setenv(k, v)
		}
	}
}

// firstExisting returns the first candidate path that exists, else the default.
func firstExisting(candidates []string, def string) string {
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return def
}

func loadFeedFile(path string) ([]Cast, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var feed persistedFeed
	if err := json.Unmarshal(raw, &feed); err != nil {
		return nil, err
	}
	return feed.Casts, nil
}

func fetchFeed(key, fid string) ([]Cast, error) {
	url := "https://api.neynar.com/v2/farcaster/feed?feed_type=following&fid=" + fid + "&limit=15&with_recasts=true"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
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
	var feed persistedFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	return feed.Casts, nil
}

func saveFeedFile(path string, casts []Cast) error {
	out, _ := json.MarshalIndent(persistedFeed{Casts: casts}, "", "  ")
	return os.WriteFile(path, out, 0644)
}

func validatePost(text string) error {
	t := strings.TrimSpace(text)
	if t == "" {
		return fmt.Errorf("empty cast")
	}
	if len([]rune(t)) > 320 {
		return fmt.Errorf("cast too long (320 max)")
	}
	return nil
}

func postCast(key, signer, text string) error {
	if signer == "" {
		return fmt.Errorf("missing SIGNER_UUID — run ./signer-setup.sh once, then approve in Warpcast")
	}
	if err := validatePost(text); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{
		"signer_uuid": signer,
		"text":        strings.TrimSpace(text),
	})
	req, _ := http.NewRequest("POST", "https://api.neynar.com/v2/farcaster/cast/", bytes.NewReader(payload))
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("neynar %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func timeAgo(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// oneLine collapses a cast into a single-line preview.
func oneLine(s string, width int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "(no text)"
	}
	return truncateRunes(s, width)
}

// wrapText word-wraps s to width, preserving paragraph breaks.
func wrapText(s string, width int) string {
	if width < 10 {
		width = 10
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			out = append(out, "")
			continue
		}
		var line []string
		cur := 0
		for _, w := range strings.Fields(para) {
			wlen := len([]rune(w))
			if cur == 0 {
				line = append(line, w)
				cur = wlen
				continue
			}
			if cur+1+wlen > width {
				out = append(out, strings.Join(line, " "))
				line = []string{w}
				cur = wlen
			} else {
				line = append(line, w)
				cur += 1 + wlen
			}
		}
		out = append(out, strings.Join(line, " "))
	}
	return strings.Join(out, "\n")
}
