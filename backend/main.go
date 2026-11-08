// Tiny Neynar -> JSON backend for low-memory laptops + Quickshell.
// No SDK, stdlib only. Keeps RAM ~10-15MB.
// Usage: NEYNAR_API_KEY=xxx go run .  (or it loads ../.env automatically)
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
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
		// Strip one layer of matching shell quotes so quoted multi-word
		// values (mnemonic) load clean in Go and source clean in sh.
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
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

	postText := flag.String("post", "", "post a cast with this text")
	parent := flag.String("parent", "", "reply target cast hash (reply mode)")
	parentFid := flag.Int64("parent-fid", 0, "reply target author fid")
	quoteHash := flag.String("quote-hash", "", "quote target cast hash (quote mode)")
	quoteFid := flag.Int64("quote-fid", 0, "quote target author fid")
	setup := flag.Bool("signer-setup", false, "one-time signer setup (create+sign+register, print approval URL)")
	flag.Parse()
	if *setup {
		signerSetup(key)
		return
	}
	if *postText != "" {
		postCast(key, *postText, *parent, *parentFid, *quoteHash, *quoteFid)
		return
	}

	// Following feed = timeline view. Limit 15 = low memory.
	fid := os.Getenv("FEED_FID")
	if fid == "" {
		fid = "10695" // @jdl
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

func postCast(key, text, parent string, parentFid int64, quoteHash string, quoteFid int64) {
	signer := os.Getenv("SIGNER_UUID")
	if signer == "" {
		fmt.Fprintln(os.Stderr, "missing SIGNER_UUID — run ./signer-setup.sh once, then approve in Warpcast")
		os.Exit(1)
	}
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "empty cast")
		os.Exit(1)
	}
	if len([]rune(text)) > 320 {
		fmt.Fprintln(os.Stderr, "cast too long (320 max)")
		os.Exit(1)
	}
	if parent != "" && quoteHash != "" {
		fmt.Fprintln(os.Stderr, "pick reply or quote, not both")
		os.Exit(1)
	}
	payload := map[string]any{
		"signer_uuid": signer,
		"text":        text,
	}
	if parent != "" {
		payload["parent"] = parent
		if parentFid != 0 {
			payload["parent_author_fid"] = parentFid
		}
	}
	if quoteHash != "" {
		payload["embeds"] = []any{map[string]any{
			"cast_id": map[string]any{"fid": quoteFid, "hash": quoteHash},
		}}
	}
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.neynar.com/v2/farcaster/cast/", bytes.NewReader(raw))
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "post error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "neynar %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}
	fmt.Println("posted:", string(body))
}
