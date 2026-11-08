// Tiny Neynar -> JSON backend for low-memory laptops + Quickshell.
// No SDK, stdlib only. Keeps RAM ~10-15MB.
// Usage: NEYNAR_API_KEY=xxx go run .  (or it loads ../.env automatically)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Cast struct {
	Hash   string `json:"hash"`
	Text   string `json:"text"`
	Author struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Fid         int    `json:"fid"`
	} `json:"author"`
	Timestamp string `json:"timestamp"`
}

type FeedResp struct {
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

func main() {
	loadDotEnv("../.env")
	loadDotEnv(".env")

	key := os.Getenv("NEYNAR_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "missing NEYNAR_API_KEY in env or ../.env")
		os.Exit(1)
	}

	// Following feed for a demo FID = timeline view. Limit 15 = low memory.
	// Works on current Neynar API (trending endpoint was removed).
	fid := os.Getenv("FEED_FID")
	if fid == "" {
		fid = "3" // dwr.eth, good demo
	}
	url := "https://api.neynar.com/v2/farcaster/feed?feed_type=following&fid=" + fid + "&limit=15&with_recasts=true"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "neynar %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var feed FeedResp
	if err := json.Unmarshal(body, &feed); err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", err)
		os.Exit(1)
	}

	// Write slim file for Quickshell FileView to watch.
	out, _ := json.MarshalIndent(feed, "", "  ")
	os.WriteFile("../feed.json", out, 0644)

	// Also print tiny TUI preview.
	for i, c := range feed.Casts {
		name := c.Author.Username
		if name == "" {
			name = fmt.Sprintf("fid:%d", c.Author.Fid)
		}
		text := strings.ReplaceAll(c.Text, "\n", " ")
		if len(text) > 120 {
			text = text[:120] + "…"
		}
		fmt.Printf("%2d. @%s: %s\n", i+1, name, text)
	}
}
